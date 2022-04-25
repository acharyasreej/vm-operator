// Copyright (c) 2020 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package virtualmachinesetresourcepolicy_test

import (
	"testing"

	. "github.com/onsi/ginkgo"

	ctrlmgr "sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/acharyasreej/vm-operator/controllers/virtualmachinesetresourcepolicy"
	ctrlContext "github.com/acharyasreej/vm-operator/pkg/context"
	providerfake "github.com/acharyasreej/vm-operator/pkg/vmprovider/fake"
	"github.com/acharyasreej/vm-operator/test/builder"
)

var intgFakeVMProvider = providerfake.NewVMProvider()

var suite = builder.NewTestSuiteForController(
	virtualmachinesetresourcepolicy.AddToManager,
	func(ctx *ctrlContext.ControllerManagerContext, _ ctrlmgr.Manager) error {
		ctx.VMProvider = intgFakeVMProvider
		return nil
	},
)

func TestVirtualMachineSetResourcePolicy(t *testing.T) {
	suite.Register(t, "VirtualMachineSetResourcePolicy controller suite", intgTests, unitTests)
}

var _ = BeforeSuite(suite.BeforeSuite)

var _ = AfterSuite(suite.AfterSuite)
