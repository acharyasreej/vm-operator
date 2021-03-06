// Copyright (c) 2020 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package virtualmachineclass_test

import (
	"testing"

	. "github.com/onsi/ginkgo"

	"github.com/acharyasreej/vm-operator/controllers/virtualmachineclass"
	"github.com/acharyasreej/vm-operator/pkg/manager"
	"github.com/acharyasreej/vm-operator/test/builder"
)

var suite = builder.NewTestSuiteForController(
	virtualmachineclass.AddToManager,
	manager.InitializeProvidersNoopFn,
)

func TestVirtualMachineClass(t *testing.T) {
	suite.Register(t, "VirtualMachineClass controller suite", intgTests, unitTests)
}

var _ = BeforeSuite(suite.BeforeSuite)

var _ = AfterSuite(suite.AfterSuite)
