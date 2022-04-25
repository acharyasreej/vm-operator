// +build integration

// Copyright (c) 2019-2020 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package integration

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/vmware/govmomi/object"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	vmoperatorv1alpha1 "github.com/acharyasreej/vm-operator-api/api/v1alpha1"

	"github.com/acharyasreej/vm-operator/pkg/record"
	"github.com/acharyasreej/vm-operator/pkg/vmprovider"
	"github.com/acharyasreej/vm-operator/pkg/vmprovider/providers/vsphere"
	"github.com/acharyasreej/vm-operator/pkg/vmprovider/providers/vsphere/config"
	"github.com/acharyasreej/vm-operator/test/builder"
	"github.com/acharyasreej/vm-operator/test/integration"
)

func createResourcePool(rpName string) (*object.ResourcePool, error) {
	rpSpec := &vmoperatorv1alpha1.ResourcePoolSpec{
		Name: rpName,
	}
	_, err := session.CreateResourcePool(context.TODO(), rpSpec)
	Expect(err).NotTo(HaveOccurred())

	return session.ChildResourcePool(context.TODO(), rpSpec.Name)
}

func createFolder(folderName string) (*object.Folder, error) {
	folderSpec := &vmoperatorv1alpha1.FolderSpec{
		Name: folderName,
	}
	_, err := session.CreateFolder(context.TODO(), folderSpec)
	Expect(err).NotTo(HaveOccurred())

	return session.ChildFolder(context.TODO(), folderSpec.Name)
}

func getSimpleVirtualMachine(name string) *vmoperatorv1alpha1.VirtualMachine {
	return &vmoperatorv1alpha1.VirtualMachine{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
}

func getVmConfigArgs(namespace, name string, imageName string) vmprovider.VMConfigArgs {
	vmClass := getVMClassInstance(name, namespace)
	vmImage := builder.DummyVirtualMachineImage(imageName)

	return vmprovider.VMConfigArgs{
		VMClass:            *vmClass,
		VMImage:            vmImage,
		ResourcePolicy:     nil,
		StorageProfileID:   "aa6d5a82-1c88-45da-85d3-3d74b91a5bad",
		ContentLibraryUUID: integration.GetContentSourceID(),
	}
}

func getVMClassInstance(vmName, namespace string) *vmoperatorv1alpha1.VirtualMachineClass {
	return &vmoperatorv1alpha1.VirtualMachineClass{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      fmt.Sprintf("%s-class", vmName),
		},
		Spec: vmoperatorv1alpha1.VirtualMachineClassSpec{
			Hardware: vmoperatorv1alpha1.VirtualMachineClassHardware{
				Cpus:   4,
				Memory: resource.MustParse("1Mi"),
			},
			Policies: vmoperatorv1alpha1.VirtualMachineClassPolicies{
				Resources: vmoperatorv1alpha1.VirtualMachineClassResources{
					Requests: vmoperatorv1alpha1.VirtualMachineResourceSpec{
						Cpu:    resource.MustParse("1000Mi"),
						Memory: resource.MustParse("100Mi"),
					},
					Limits: vmoperatorv1alpha1.VirtualMachineResourceSpec{
						Cpu:    resource.MustParse("2000Mi"),
						Memory: resource.MustParse("200Mi"),
					},
				},
			},
		},
	}
}

func getVirtualMachineInstance(name, namespace, imageName, className string) *vmoperatorv1alpha1.VirtualMachine {
	return &vmoperatorv1alpha1.VirtualMachine{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
		Spec: vmoperatorv1alpha1.VirtualMachineSpec{
			ImageName:  imageName,
			ClassName:  className,
			PowerState: vmoperatorv1alpha1.VirtualMachinePoweredOn,
			Ports:      []vmoperatorv1alpha1.VirtualMachinePort{},
		},
	}
}

var _ = Describe("VMProvider Inventory Tests", func() {
	Context("When using inventory", func() {
		It("should support controller like workflow", func() {
			vmNamespace := integration.DefaultNamespace
			vmName := "test-vm-vmp-invt-deploy"
			storageProfileId := "aa6d5a82-1c88-45da-85d3-3d74b91a5bad"

			vmMetadata := vmprovider.VMMetadata{
				Transport: vmoperatorv1alpha1.VirtualMachineMetadataOvfEnvTransport,
			}
			imageName := "DC0_H0_VM0" // Default govcsim image name
			vmClass := getVMClassInstance(vmName, vmNamespace)
			vm := getVirtualMachineInstance(vmName, vmNamespace, imageName, vmClass.Name)
			vmImage := builder.DummyVirtualMachineImage(imageName)
			Expect(vm.Status.BiosUUID).Should(BeEmpty())
			Expect(vm.Status.InstanceUUID).Should(BeEmpty())

			exists, err := vmProvider.DoesVirtualMachineExist(ctx, vm)
			Expect(err).ToNot(HaveOccurred())
			Expect(exists).To(BeFalse())

			vmConfigArgs := vmprovider.VMConfigArgs{
				VMClass:          *vmClass,
				VMImage:          vmImage,
				VMMetadata:       vmMetadata,
				StorageProfileID: storageProfileId,
			}
			err = vmProvider.CreateVirtualMachine(context.TODO(), vm, vmConfigArgs)
			Expect(err).NotTo(HaveOccurred())

			// Update Virtual Machine to Reconfigure with VM Class config
			err = vmProvider.UpdateVirtualMachine(context.TODO(), vm, vmConfigArgs)
			Expect(err).NotTo(HaveOccurred())
			Expect(vm.Status.PowerState).Should(Equal(vmoperatorv1alpha1.VirtualMachinePoweredOn))
			Expect(vm.Status.BiosUUID).ShouldNot(BeEmpty())
			Expect(vm.Status.InstanceUUID).ShouldNot(BeEmpty())

			exists, err = vmProvider.DoesVirtualMachineExist(ctx, vm)
			Expect(err).ToNot(HaveOccurred())
			Expect(exists).To(BeTrue())

			vm.Spec.PowerState = vmoperatorv1alpha1.VirtualMachinePoweredOn
			err = vmProvider.UpdateVirtualMachine(context.TODO(), vm, vmConfigArgs)
			Expect(err).ToNot(HaveOccurred())
			Expect(vm.Status.PowerState).To(Equal(vmoperatorv1alpha1.VirtualMachinePoweredOn))
			Expect(vm.Status.Host).ToNot(BeEmpty())
			Expect(vm.Status.UniqueID).ToNot(BeEmpty())
			Expect(vm.Status.BiosUUID).ToNot(BeEmpty())
			Expect(vm.Status.InstanceUUID).ToNot(BeEmpty())

			vm.Spec.PowerState = vmoperatorv1alpha1.VirtualMachinePoweredOff
			err = vmProvider.UpdateVirtualMachine(context.TODO(), vm, vmConfigArgs)
			Expect(err).ToNot(HaveOccurred())
			Expect(vm.Status.PowerState).To(Equal(vmoperatorv1alpha1.VirtualMachinePoweredOff))

			err = vmProvider.DeleteVirtualMachine(ctx, vm)
			Expect(err).ToNot(HaveOccurred())
		})
	})
})

var _ = Describe("VMProvider Tests", func() {
	var (
		recorder record.Recorder
	)

	BeforeEach(func() {
		recorder, _ = builder.NewFakeRecorder()
	})

	Context("When using Content Library", func() {
		var vmProvider vmprovider.VirtualMachineProviderInterface
		var err error
		vmNamespace := integration.DefaultNamespace
		vmName := "test-vm-vmp-deploy"
		storageProfileId := "aa6d5a82-1c88-45da-85d3-3d74b91a5bad"

		BeforeEach(func() {
			err = config.InstallVSphereVMProviderConfig(k8sClient, integration.DefaultNamespace,
				integration.NewIntegrationVMOperatorConfig(vcSim.IP, vcSim.Port),
				integration.SecretName)
			Expect(err).NotTo(HaveOccurred())

			vmProvider = vsphere.NewVSphereVMProviderFromClient(k8sClient, recorder)

			// Instruction to vcsim to give the VM an IP address, otherwise CreateVirtualMachine fails
			// BMV: Not true anymore, and we can't set this via ExtraConfig transport anyways.
			testIP := "10.0.0.1"
			vmMetadata := vmprovider.VMMetadata{
				Data:      map[string]string{"SET.guest.ipAddress": testIP},
				Transport: vmoperatorv1alpha1.VirtualMachineMetadataExtraConfigTransport,
			}
			imageName := integration.IntegrationContentLibraryItemName
			vmClass := getVMClassInstance(vmName, vmNamespace)
			vm := getVirtualMachineInstance(vmName, vmNamespace, imageName, vmClass.Name)
			vmImage := builder.DummyVirtualMachineImage(imageName)
			Expect(vm.Status.BiosUUID).Should(BeEmpty())
			Expect(vm.Status.InstanceUUID).Should(BeEmpty())

			exists, err := vmProvider.DoesVirtualMachineExist(ctx, vm)
			Expect(err).ToNot(HaveOccurred())
			Expect(exists).To(BeFalse())

			// CreateVirtualMachine from CL
			vmConfigArgs := vmprovider.VMConfigArgs{
				VMClass:            *vmClass,
				VMImage:            vmImage,
				VMMetadata:         vmMetadata,
				StorageProfileID:   storageProfileId,
				ContentLibraryUUID: integration.GetContentSourceID(),
			}
			err = vmProvider.CreateVirtualMachine(context.TODO(), vm, vmConfigArgs)
			Expect(err).NotTo(HaveOccurred())

			// Update Virtual Machine to Reconfigure with VM Class config
			err = vmProvider.UpdateVirtualMachine(context.TODO(), vm, vmConfigArgs)
			Expect(err).NotTo(HaveOccurred())
			//Expect(vm.Status.VmIp).Should(Equal(testIP))
			Expect(vm.Status.PowerState).Should(Equal(vmoperatorv1alpha1.VirtualMachinePoweredOn))
			Expect(vm.Status.BiosUUID).ShouldNot(BeEmpty())
			Expect(vm.Status.InstanceUUID).ShouldNot(BeEmpty())
		})

		It("should work", func() {
			// Everything done in the BeforeEach().
		})

		// DWB: Disabling this test until I work with Doug M. to determine why there is a FileAlreadyExists error being
		// emitted by Govmomi (I suspect from simulator/virtual_machine.go
		XIt("2 VMs with the same name should be created in different namespaces", func() {
			sameVmName := "same-vm"
			vmNamespace1 := vmNamespace + "-1"
			vmNamespace2 := vmNamespace + "-2"

			err := k8sClient.Create(ctx, &v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: vmNamespace1}})
			Expect(err).NotTo(HaveOccurred())

			err = k8sClient.Create(ctx, &v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: vmNamespace2}})
			Expect(err).NotTo(HaveOccurred())

			folder1, err := createFolder(vmNamespace1)
			Expect(err).NotTo(HaveOccurred())

			rp1, err := createResourcePool(vmNamespace1)
			Expect(err).NotTo(HaveOccurred())

			folder2, err := createFolder(vmNamespace2)
			Expect(err).NotTo(HaveOccurred())

			rp2, err := createResourcePool(vmNamespace2)
			Expect(err).NotTo(HaveOccurred())

			resourcePolicy1 := &vmoperatorv1alpha1.VirtualMachineSetResourcePolicy{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: vmNamespace1,
					Name:      sameVmName,
				},
				Spec: vmoperatorv1alpha1.VirtualMachineSetResourcePolicySpec{
					ResourcePool: vmoperatorv1alpha1.ResourcePoolSpec{
						Name:         rp1.Name(),
						Reservations: vmoperatorv1alpha1.VirtualMachineResourceSpec{},
						Limits:       vmoperatorv1alpha1.VirtualMachineResourceSpec{},
					},
					Folder: vmoperatorv1alpha1.FolderSpec{
						Name: folder1.Name(),
					},
				},
			}
			Expect(vmProvider.CreateOrUpdateVirtualMachineSetResourcePolicy(context.TODO(), resourcePolicy1)).To(Succeed())

			resourcePolicy2 := &vmoperatorv1alpha1.VirtualMachineSetResourcePolicy{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: vmNamespace2,
					Name:      sameVmName,
				},
				Spec: vmoperatorv1alpha1.VirtualMachineSetResourcePolicySpec{
					ResourcePool: vmoperatorv1alpha1.ResourcePoolSpec{
						Name:         rp2.Name(),
						Reservations: vmoperatorv1alpha1.VirtualMachineResourceSpec{},
						Limits:       vmoperatorv1alpha1.VirtualMachineResourceSpec{},
					},
					Folder: vmoperatorv1alpha1.FolderSpec{
						Name: folder2.Name(),
					},
				},
			}
			Expect(vmProvider.CreateOrUpdateVirtualMachineSetResourcePolicy(context.TODO(), resourcePolicy2)).To(Succeed())
			imageName := integration.IntegrationContentLibraryItemName

			vmImage := builder.DummyVirtualMachineImage(imageName)

			vmConfigArgs1 := vmprovider.VMConfigArgs{
				VMClass:          *getVMClassInstance(sameVmName, vmNamespace1),
				VMImage:          vmImage,
				ResourcePolicy:   resourcePolicy1,
				StorageProfileID: "aa6d5a82-1c88-45da-85d3-3d74b91a5bad",
			}

			vmConfigArgs2 := vmprovider.VMConfigArgs{
				VMClass:          *getVMClassInstance(sameVmName, vmNamespace2),
				VMImage:          vmImage,
				ResourcePolicy:   resourcePolicy2,
				StorageProfileID: "aa6d5a82-1c88-45da-85d3-3d74b91a5bad",
			}

			vm1 := &vmoperatorv1alpha1.VirtualMachine{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: vmNamespace1,
					Name:      sameVmName,
				},
				Spec: vmoperatorv1alpha1.VirtualMachineSpec{
					ImageName:          imageName,
					ClassName:          vmConfigArgs1.VMClass.Name,
					PowerState:         vmoperatorv1alpha1.VirtualMachinePoweredOn,
					Ports:              []vmoperatorv1alpha1.VirtualMachinePort{},
					ResourcePolicyName: resourcePolicy1.Name,
				},
			}

			// CreateVirtualMachine from CL
			err = vmProvider.CreateVirtualMachine(context.TODO(), vm1, vmConfigArgs1)
			Expect(err).NotTo(HaveOccurred())

			vm2 := &vmoperatorv1alpha1.VirtualMachine{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: vmNamespace2,
					Name:      sameVmName,
				},
				Spec: vmoperatorv1alpha1.VirtualMachineSpec{
					ImageName:          imageName,
					ClassName:          vmConfigArgs2.VMClass.Name,
					PowerState:         vmoperatorv1alpha1.VirtualMachinePoweredOn,
					Ports:              []vmoperatorv1alpha1.VirtualMachinePort{},
					ResourcePolicyName: resourcePolicy2.Name,
				},
			}
			err = vmProvider.CreateVirtualMachine(context.TODO(), vm2, vmConfigArgs2)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("Compute CPU Min Frequency in the Cluster", func() {
		It("reconfigure and power on without errors", func() {
			vmProvider := vsphere.NewVSphereVMProviderFromClient(k8sClient, recorder)
			vcClient, err := vmProvider.(vsphere.VSphereVMProviderGetSessionHack).GetClient(ctx)
			Expect(vcClient).ToNot(BeNil())
			Expect(err).NotTo(HaveOccurred())
			err = vmProvider.ComputeClusterCPUMinFrequency(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(session.GetCPUMinMHzInCluster()).Should(BeNumerically(">", 0))
		})
	})
})
