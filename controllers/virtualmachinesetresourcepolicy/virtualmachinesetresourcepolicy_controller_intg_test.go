// Copyright (c) 2019-2020 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package virtualmachinesetresourcepolicy_test

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	vmopv1alpha1 "github.com/acharyasreej/vm-operator-api/api/v1alpha1"

	"github.com/acharyasreej/vm-operator/test/builder"
)

func intgTests() {
	var (
		ctx *builder.IntegrationTestContext

		resourcePolicy    *vmopv1alpha1.VirtualMachineSetResourcePolicy
		resourcePolicyKey client.ObjectKey
	)

	BeforeEach(func() {
		ctx = suite.NewIntegrationTestContext()

		resourcePolicy = &vmopv1alpha1.VirtualMachineSetResourcePolicy{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ctx.Namespace,
				Name:      "dummy-vm-policy",
			},
			Spec: vmopv1alpha1.VirtualMachineSetResourcePolicySpec{},
		}

		resourcePolicyKey = client.ObjectKey{Namespace: resourcePolicy.Namespace, Name: resourcePolicy.Name}
	})

	AfterEach(func() {
		ctx.AfterEach()
		ctx = nil
		intgFakeVMProvider.Reset()
	})

	getResourcePolicy := func(ctx *builder.IntegrationTestContext, objKey client.ObjectKey) *vmopv1alpha1.VirtualMachineSetResourcePolicy {
		rp := &vmopv1alpha1.VirtualMachineSetResourcePolicy{}
		if err := ctx.Client.Get(ctx, objKey, rp); err != nil {
			return nil
		}
		return rp
	}

	waitForResourcePolicyFinalizer := func(ctx *builder.IntegrationTestContext, objKey client.ObjectKey) {
		Eventually(func() []string {
			if rp := getResourcePolicy(ctx, objKey); rp != nil {
				return rp.GetFinalizers()
			}
			return nil
		}).Should(ContainElement(finalizer), "waiting for VirtualMachineSetResourcePolicy finalizer")
	}

	Context("Reconcile", func() {
		var called bool

		BeforeEach(func() {
			intgFakeVMProvider.Lock()
			intgFakeVMProvider.CreateOrUpdateVirtualMachineSetResourcePolicyFn = func(_ context.Context, _ *vmopv1alpha1.VirtualMachineSetResourcePolicy) error {
				called = true
				return nil
			}
			intgFakeVMProvider.Unlock()
		})

		It("Reconciles after VirtualMachineSetResourcePolicy creation", func() {
			Expect(ctx.Client.Create(ctx, resourcePolicy)).To(Succeed())

			By("VirtualMachineSetResourcePolicy should have finalizer added", func() {
				waitForResourcePolicyFinalizer(ctx, resourcePolicyKey)
			})

			By("Create policy should be called", func() {
				Eventually(called).Should(BeTrue())
			})

			By("Deleting the VirtualMachineSetResourcePolicy", func() {
				err := ctx.Client.Delete(ctx, resourcePolicy)
				Expect(err).ToNot(HaveOccurred())
			})

			By("VirtualMachineSetResourcePolicy should have finalizer removed", func() {
				Eventually(func() []string {
					if rp := getResourcePolicy(ctx, resourcePolicyKey); rp != nil {
						return rp.GetFinalizers()
					}
					return nil
				}).ShouldNot(ContainElement(finalizer))
			})
		})
	})
}
