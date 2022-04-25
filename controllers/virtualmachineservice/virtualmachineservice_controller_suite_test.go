// Copyright (c) 2019-2020 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package virtualmachineservice_test

import (
	"testing"

	. "github.com/onsi/ginkgo"

	"github.com/acharyasreej/vm-operator/controllers/virtualmachineservice"
	"github.com/acharyasreej/vm-operator/pkg/manager"
	"github.com/acharyasreej/vm-operator/test/builder"
)

var suite = builder.NewTestSuiteForController(
	virtualmachineservice.AddToManager,
	manager.InitializeProvidersNoopFn,
)

func TestVirtualMachineService(t *testing.T) {
	suite.Register(t, "VirtualMachineService controller suite", intgTests, unitTests)
}

var _ = BeforeSuite(suite.BeforeSuite)

var _ = AfterSuite(suite.AfterSuite)
