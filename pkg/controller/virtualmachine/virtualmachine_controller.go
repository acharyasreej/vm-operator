/* **********************************************************
 * Copyright 2019 VMware, Inc.  All rights reserved. -- VMware Confidential
 * **********************************************************/

package virtualmachine

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	vimtypes "github.com/vmware/govmomi/vim25/types"
	v1 "k8s.io/api/core/v1"
	storagetypev1 "k8s.io/api/storage/v1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/vmware-tanzu/vm-operator/pkg"
	"github.com/vmware-tanzu/vm-operator/pkg/apis/vmoperator"
	"github.com/vmware-tanzu/vm-operator/pkg/apis/vmoperator/v1alpha1"
	vmoperatorv1alpha1 "github.com/vmware-tanzu/vm-operator/pkg/apis/vmoperator/v1alpha1"
	"github.com/vmware-tanzu/vm-operator/pkg/controller/common"
	"github.com/vmware-tanzu/vm-operator/pkg/controller/common/record"
	volumeproviders "github.com/vmware-tanzu/vm-operator/pkg/controller/virtualmachine/providers"
	"github.com/vmware-tanzu/vm-operator/pkg/lib"
	"github.com/vmware-tanzu/vm-operator/pkg/vmprovider"
	cnsv1alpha1 "gitlab.eng.vmware.com/hatchway/vsphere-csi-driver/pkg/syncer/cnsoperator/apis/cnsnodevmattachment/v1alpha1"
)

const (
	OpCreate = "CreateVM"
	OpDelete = "DeleteVM"
	OpUpdate = "UpdateVM"

	ControllerName                 = "virtualmachine-controller"
	storageResourceQuotaStrPattern = ".storageclass.storage.k8s.io/"

	// We should add more uniqueness to the OpId to prevent the collision incurred by different vm-operator to aid debugging at VPXD.
	opIdRandomLen = 8
)

var log = logf.Log.WithName(ControllerName)

// Add creates a new VirtualMachine Controller and adds it to the Manager with default RBAC. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	// Get provider registered in the manager's main()
	provider := vmprovider.GetVmProviderOrDie()

	return &ReconcileVirtualMachine{
		Client:         mgr.GetClient(),
		scheme:         mgr.GetScheme(),
		vmProvider:     provider,
		volumeProvider: volumeproviders.CnsVolumeProvider(mgr.GetClient()),
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New(ControllerName, mgr, controller.Options{Reconciler: r, MaxConcurrentReconciles: common.GetMaxReconcileNum()})
	if err != nil {
		return err
	}

	// Watch for changes to VirtualMachine
	err = c.Watch(&source.Kind{Type: &vmoperatorv1alpha1.VirtualMachine{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// Watch for changes for CnsNodeVmAttachment, and enqueue VirtualMachine which is the owner of CnsNodeVmAttachment
	err = c.Watch(&source.Kind{Type: &cnsv1alpha1.CnsNodeVmAttachment{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &vmoperatorv1alpha1.VirtualMachine{},
	})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileVirtualMachine{}

// ReconcileVirtualMachine reconciles a VirtualMachine object
type ReconcileVirtualMachine struct {
	client.Client
	scheme         *runtime.Scheme
	vmProvider     vmprovider.VirtualMachineProviderInterface
	volumeProvider volumeproviders.VolumeProviderInterface
}

// Reconcile reads that state of the cluster for a VirtualMachine object and makes changes based on the state read
// and what is in the VirtualMachine.Spec
// Automatically generate RBAC rules to allow the Controller to read and write Deployments
// +kubebuilder:rbac:groups=vmoperator.vmware.com,resources=virtualmachines,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=vmoperator.vmware.com,resources=virtualmachines/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=vmoperator.vmware.com,resources=virtualmachineclasses,verbs=get;list
// +kubebuilder:rbac:groups=cns.vmware.com,resources=cnsnodevmattachments,verbs=create;delete;get;list;watch;patch;update
// +kubebuilder:rbac:groups=cns.vmware.com,resources=cnsnodevmattachments/status,verbs=get;list
// +kubebuilder:rbac:groups=storage.k8s.io,resources=storageclasses,verbs=get;list;watch
// +kubebuilder:rbac:groups=vmware.com,resources=virtualnetworkinterfaces;virtualnetworkinterfaces/status,verbs=create;get;list;patch;delete;watch;update
// +kubebuilder:rbac:groups="",resources=events;configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=resourcequotas,verbs=get;list
// +kubebuilder:rbac:groups="",resources=configmaps/status,verbs=get;update;patch
func (r *ReconcileVirtualMachine) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	ctx := context.Background()

	// Fetch the VirtualMachine instance
	instance := &vmoperatorv1alpha1.VirtualMachine{}
	err := r.Get(ctx, request.NamespacedName, instance)
	if err != nil {
		if apiErrors.IsNotFound(err) {
			// Object not found, return.  Created objects are automatically garbage collected.
			// For additional cleanup logic use finalizers.
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	// Use whatever clusterID is returned, even the empty string.
	clusterID, _ := r.vmProvider.GetClusterID(ctx, request.Namespace)
	ctx = context.WithValue(ctx, vimtypes.ID{}, "vmoperator-"+ControllerName+"-"+clusterID+"-"+instance.Name)

	if !instance.ObjectMeta.DeletionTimestamp.IsZero() {
		err = r.reconcileDelete(ctx, instance)
		return reconcile.Result{}, err
	}

	if err := r.reconcileNormal(ctx, instance); err != nil {
		log.Error(err, "Failed to reconcile VirtualMachine", "name", instance.NamespacedName())
		return reconcile.Result{}, err
	}

	return reconcile.Result{RequeueAfter: requeuDelay(ctx, instance)}, nil
}

// Determine if we should request a non-zero requeue delay in order to trigger a non-rate limited reconcile
// at some point in the future.  Use this delay-based reconcile to trigger a specific reconcile to discovery the VM IP
// address rather than relying on the resync period to do.
//
// TODO: It would be much prefereable to determine that a non-error resync is required at the source of the determination that
// TODO: the VM IP isn't available rather than up here in the reconcile loop.  However, in the interest of time, we are making
// TODO: this determination here and will have to refactor at some later date.
func requeuDelay(ctx context.Context, vm *vmoperatorv1alpha1.VirtualMachine) time.Duration {
	if vm.Status.VmIp == "" && vm.Status.PowerState == vmoperatorv1alpha1.VirtualMachinePoweredOn {
		return 10 * time.Second
	}

	return 0
}

func (r *ReconcileVirtualMachine) deleteVm(ctx context.Context, vm *vmoperatorv1alpha1.VirtualMachine) (err error) {
	defer record.EmitEvent(vm, OpDelete, &err, false)

	// TODO The length limit of OpId is 256, we will implement a function to detects if the total length exceeds
	//   the limit and then “smartly” trims each component of the op id to an abbreviated version
	// BMV: Push into provider
	opId := ctx.Value(vimtypes.ID{}).(string)
	ctx = context.WithValue(ctx, vimtypes.ID{}, opId+"-delete-"+common.RandomString(opIdRandomLen))

	err = r.vmProvider.DeleteVirtualMachine(ctx, vm)
	if err != nil {
		if apiErrors.IsNotFound(err) {
			log.Info("To be deleted VirtualMachine was not found", "name", vm.NamespacedName())
			return nil
		}
		log.Error(err, "Failed to delete VirtualMachine", "name", vm.NamespacedName())
		return err
	}

	vm.Status.Phase = vmoperatorv1alpha1.Deleted
	log.V(4).Info("Deleted VirtualMachine", "name", vm.NamespacedName())

	return nil
}

func (r *ReconcileVirtualMachine) reconcileDelete(ctx context.Context, vm *vmoperatorv1alpha1.VirtualMachine) error {
	log.Info("Reconciling VirtualMachine Deletion", "name", vm.NamespacedName())
	defer func() {
		log.Info("Finished Reconciling VirtualMachine Deletion", "name", vm.NamespacedName())
	}()

	const finalizerName = vmoperator.VirtualMachineFinalizer

	if lib.ContainsString(vm.ObjectMeta.Finalizers, finalizerName) {
		if err := r.deleteVm(ctx, vm); err != nil {
			return err
		}

		vm.ObjectMeta.Finalizers = lib.RemoveString(vm.ObjectMeta.Finalizers, finalizerName)
		if err := r.Update(ctx, vm); err != nil {
			return err
		}
	}

	// Our finalizer has finished, so the reconciler can do nothing.
	return nil
}

func (r *ReconcileVirtualMachine) reconcileVolumes(ctx context.Context, vm *vmoperatorv1alpha1.VirtualMachine,
	volumesToAdd map[client.ObjectKey]bool, volumesToDelete map[client.ObjectKey]bool) volumeproviders.CombinedVolumeOpsErrors {

	volumeOpsErrs := volumeproviders.CombinedVolumeOpsErrors{}

	// Create CnsNodeVMAttachments on demand if VM has been reconciled properly
	volumeOpsErrs.Append(r.volumeProvider.AttachVolumes(ctx, vm, volumesToAdd))

	// Delete CnsNodeVMAttachments on demand if VM has been reconciled properly
	volumeOpsErrs.Append(r.volumeProvider.DetachVolumes(ctx, vm, volumesToDelete))

	// Update the VirtualMachineVolumeStatus based on the status of respective CnsNodeVmAttachment instance
	volumeOpsErrs.Append(r.volumeProvider.UpdateVmVolumesStatus(ctx, vm))

	// 1. AttachVolumes() returns error when it fails to create CnsNodeVmAttachment instances (partially or completely)
	//  1.1 If the attach operation [succeeds partially | fails completely], there is no reason to return without
	// calling(r.Status().Update(ctx, vm)) to update the status for the succeeded set of volumes.
	//
	// 2. DetachVolumes() returns error when it fails to delete CnsNodeVmAttachment instances (partially or completely)
	//  2.1 If the detach operation [succeeds partially | fails completely], there is no reason to return without
	// calling (r.Status().Update(ctx, vm)) to update the status for the succeeded set of volumes.
	//
	// 3. UpdateVmVolumesStatus() does not call client.Status().Update(), it only modifies the vm object by updating the vm.Status.Volumes.
	// It returns error when it fails to get the CnsNodeVmAttachment instance which is supposed to exist.
	//  3.1 If update volumes status succeeds partially, there is no reason to return without calling (r.Status().Update(ctx, vm))

	return volumeOpsErrs
}

// Process a level trigger for this VM: create if it doesn't exist otherwise update the existing VM.
func (r *ReconcileVirtualMachine) reconcileNormal(ctx context.Context, vm *vmoperatorv1alpha1.VirtualMachine) error {
	log.Info("Reconciling VirtualMachine", "name", vm.NamespacedName())
	defer func() {
		log.Info("Finished Reconciling VirtualMachine", "name", vm.NamespacedName())
	}()

	exists, err := r.vmProvider.DoesVirtualMachineExist(ctx, vm)
	if err != nil {
		log.Error(err, "Failed to check if VirtualMachine exists from provider", "name", vm.NamespacedName())
		return err
	}

	// Before the VM being created or updated, figure out the Volumes[] VirtualMachineVolumes change.
	// This step determines what CnsNodeVmAttachments need to be created or deleted
	// BMV: Push into provider
	volumesToAdd, volumesToDelete := volumeproviders.GetVmVolumesToProcess(vm)

	if !exists {
		err = r.createVm(ctx, vm)
		if err != nil {
			return err
		}
	}

	err = r.updateVm(ctx, vm)
	if err != nil {
		return err
	}

	volumeOpsErrs := r.reconcileVolumes(ctx, vm, volumesToAdd, volumesToDelete)

	if err := r.Status().Update(ctx, vm); err != nil {
		log.Error(err, "Failed to update VirtualMachine status", "name", vm.NamespacedName())
		return err
	}

	if volumeOpsErrs.HasOccurred() {
		return volumeOpsErrs
	}
	return nil
}

func (r *ReconcileVirtualMachine) validateStorageClass(ctx context.Context, namespace, scName string) error {
	resourceQuotas := &v1.ResourceQuotaList{}
	err := r.List(ctx, client.InNamespace(namespace), resourceQuotas)
	if err != nil {
		log.Error(err, "Failed to list ResourceQuotas", "namespace", namespace)
		return err
	}

	if len(resourceQuotas.Items) == 0 {
		return fmt.Errorf("no ResourceQuotas assigned to namespace '%s'", namespace)
	}

	for _, resourceQuota := range resourceQuotas.Items {
		for resourceName := range resourceQuota.Spec.Hard {
			resourceNameStr := resourceName.String()
			if !strings.Contains(resourceNameStr, storageResourceQuotaStrPattern) {
				continue
			}
			// BMV: Match prefix 'scName + storageResourceQuotaStrPattern'?
			scNameFromRQ := strings.Split(resourceNameStr, storageResourceQuotaStrPattern)[0]
			if scName == scNameFromRQ {
				return nil
			}
		}
	}

	return fmt.Errorf("StorageClass '%s' is not assigned to any ResourceQuotas in namespace '%s'", scName, namespace)
}

func (r *ReconcileVirtualMachine) lookupStoragePolicyID(ctx context.Context, namespace, storageClassName string) (string, error) {
	err := r.validateStorageClass(ctx, namespace, storageClassName)
	if err != nil {
		return "", err
	}

	sc := &storagetypev1.StorageClass{}
	err = r.Client.Get(ctx, client.ObjectKey{Name: storageClassName}, sc)
	if err != nil {
		log.Error(err, "Failed to get StorageClass", "storageClassName", storageClassName)
		return "", err
	}

	return sc.Parameters["storagePolicyID"], nil
}

func (r *ReconcileVirtualMachine) createVm(ctx context.Context, vm *vmoperatorv1alpha1.VirtualMachine) (err error) {
	defer record.EmitEvent(vm, OpCreate, &err, false)

	// TODO The length limit of OpId is 256, we will implement a function to detects if the total length
	//   exceeds the limit and then "smartly" trims each component of the op id to an abbreviated version.
	// BMV: Push into provider
	opId := ctx.Value(vimtypes.ID{}).(string)
	ctx = context.WithValue(ctx, vimtypes.ID{}, opId+"-create-"+common.RandomString(opIdRandomLen))

	vmClass := &vmoperatorv1alpha1.VirtualMachineClass{}
	err = r.Get(ctx, client.ObjectKey{Name: vm.Spec.ClassName}, vmClass)
	if err != nil {
		log.Error(err, "Failed to get VirtualMachineClass for VirtualMachine",
			"vmName", vm.NamespacedName(), "class", vm.Spec.ClassName)
		return err
	}

	var vmMetadata vmprovider.VirtualMachineMetadata
	if metadata := vm.Spec.VmMetadata; metadata != nil {
		configMap := &v1.ConfigMap{}
		err := r.Get(ctx, types.NamespacedName{Name: metadata.ConfigMapName, Namespace: vm.Namespace}, configMap)
		if err != nil {
			log.Error(err, "Failed to get VirtualMachineMetadata ConfigMap",
				"vmName", vm.NamespacedName(), "configMapName", metadata.ConfigMapName)
			return err
		}

		vmMetadata = configMap.Data
	}

	var resourcePolicy *v1alpha1.VirtualMachineSetResourcePolicy
	if policyName := vm.Spec.ResourcePolicyName; policyName != "" {
		resourcePolicy = &vmoperatorv1alpha1.VirtualMachineSetResourcePolicy{}
		err = r.Get(ctx, client.ObjectKey{Name: policyName, Namespace: vm.Namespace}, resourcePolicy)
		if err != nil {
			log.Error(err, "Failed to get VirtualMachineSetResourcePolicy",
				"vmName", vm.NamespacedName(), "resourcePolicyName", policyName)
			return err
		}

		// Make sure that the corresponding entities (RP and Folder) are created on the infra provider before
		// creating the VM. Requeue if the ResourcePool and Folders are not yet created for this ResourcePolicy
		rpReady, err := r.vmProvider.DoesVirtualMachineSetResourcePolicyExist(ctx, resourcePolicy)
		if err != nil {
			log.Error(err, "Failed to check if VirtualMachineSetResourcePolicy exists")
			return err
		}
		if !rpReady {
			return errors.New("VirtualMachineSetResourcePolicy not actualized yet. Failing VirtualMachine reconcile")
		}
	}

	var storagePolicyID string
	if storageClass := vm.Spec.StorageClass; storageClass != "" {
		storagePolicyID, err = r.lookupStoragePolicyID(ctx, vm.Namespace, storageClass)
		if err != nil {
			return err
		}
	}

	vmConfigArgs := vmprovider.VmConfigArgs{
		VmClass:          *vmClass,
		VmMetadata:       vmMetadata,
		ResourcePolicy:   resourcePolicy,
		StorageProfileID: storagePolicyID,
	}
	err = r.vmProvider.CreateVirtualMachine(ctx, vm, vmConfigArgs)
	if err != nil {
		log.Error(err, "Provider failed to create VirtualMachine", "name", vm.NamespacedName())
		return err
	}

	return nil
}

func (r *ReconcileVirtualMachine) updateVm(ctx context.Context, vm *vmoperatorv1alpha1.VirtualMachine) (err error) {

	// TODO The length limit of OpId is 256, we will implement a function to detects if the total length
	//   exceeds the limit and then "smartly" trims each component of the op id to an abbreviated version.
	// BMV: Push into provider
	opId := ctx.Value(vimtypes.ID{}).(string)
	ctx = context.WithValue(ctx, vimtypes.ID{}, opId+"-update-"+common.RandomString(opIdRandomLen))

	vmClass := &vmoperatorv1alpha1.VirtualMachineClass{}
	err = r.Get(ctx, client.ObjectKey{Name: vm.Spec.ClassName}, vmClass)
	if err != nil {
		log.Error(err, "Failed to get VirtualMachineClass for VirtualMachine",
			"vmName", vm.NamespacedName(), "class", vm.Spec.ClassName)
		return err
	}

	var vmMetadata vmprovider.VirtualMachineMetadata
	if metadata := vm.Spec.VmMetadata; metadata != nil {
		configMap := &v1.ConfigMap{}
		err := r.Get(ctx, types.NamespacedName{Name: metadata.ConfigMapName, Namespace: vm.Namespace}, configMap)
		if err != nil {
			record.EmitEvent(vm, OpUpdate, &err, false)
			log.Error(err, "Failed to get VirtualMachineMetadata ConfigMap",
				"vmName", vm.NamespacedName(), "configMapName", metadata.ConfigMapName)
			return err
		}

		vmMetadata = configMap.Data
	}

	pkg.AddAnnotations(&vm.ObjectMeta)

	var resourcePolicy *v1alpha1.VirtualMachineSetResourcePolicy
	if policyName := vm.Spec.ResourcePolicyName; policyName != "" {
		resourcePolicy = &vmoperatorv1alpha1.VirtualMachineSetResourcePolicy{}
		err = r.Get(ctx, client.ObjectKey{Name: policyName, Namespace: vm.Namespace}, resourcePolicy)
		if err != nil {
			log.Error(err, "Failed to get VirtualMachineSetResourcePolicy",
				"vmName", vm.NamespacedName(), "resourcePolicyName", policyName)
			return err
		}
	}
	vmConfigArgs := vmprovider.VmConfigArgs{
		VmClass:        *vmClass,
		VmMetadata:     vmMetadata,
		ResourcePolicy: resourcePolicy,
	}
	err = r.vmProvider.UpdateVirtualMachine(ctx, vm, vmConfigArgs)

	if err != nil {
		log.Error(err, "Provider failed to update VirtualMachine", "name", vm.NamespacedName())
		return err
	}

	return nil
}
