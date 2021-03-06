// Copyright (c) 2019-2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package virtualmachineimage_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	vmopv1alpha1 "github.com/acharyasreej/vm-operator-api/api/v1alpha1"

	"github.com/acharyasreej/vm-operator/controllers/virtualmachineimage"
	"github.com/acharyasreej/vm-operator/test/builder"
)

func unitTests() {
	Describe("Invoking VirtualMachineImage controller tests", unitTestsReconcile)
}

func unitTestsReconcile() {
	var (
		initObjects []client.Object
		ctx         *builder.UnitTestContextForController

		reconciler *virtualmachineimage.Reconciler
		vmImage    *vmopv1alpha1.VirtualMachineImage
	)

	BeforeEach(func() {
		vmImage = &vmopv1alpha1.VirtualMachineImage{
			ObjectMeta: metav1.ObjectMeta{
				Name: "dummy-vmclass",
			},
		}
	})

	JustBeforeEach(func() {
		ctx = suite.NewUnitTestContextForController(initObjects...)
		reconciler = virtualmachineimage.NewReconciler(
			ctx.Client,
			ctx.Logger,
		)
	})

	Context("ReconcileNormal", func() {
		BeforeEach(func() {
			initObjects = append(initObjects, vmImage)
		})

		When("NoOp", func() {
			It("returns success", func() {
				err := reconciler.ReconcileNormal(ctx.Context)
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})
}
