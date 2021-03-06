// Copyright (c) 2020 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package virtualmachineimage_test

import (
	"testing"

	. "github.com/onsi/ginkgo"

	"github.com/acharyasreej/vm-operator/controllers/virtualmachineimage"
	"github.com/acharyasreej/vm-operator/pkg/manager"
	"github.com/acharyasreej/vm-operator/test/builder"
)

var suite = builder.NewTestSuiteForController(
	virtualmachineimage.AddToManager,
	manager.InitializeProvidersNoopFn,
)

func TestVirtualMachineImage(t *testing.T) {
	suite.Register(t, "VirtualMachineImage controller suite", intgTests, unitTests)
}

var _ = BeforeSuite(suite.BeforeSuite)

var _ = AfterSuite(suite.AfterSuite)
