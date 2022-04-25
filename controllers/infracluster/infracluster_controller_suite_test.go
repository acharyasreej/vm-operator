// Copyright (c) 2020 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package infracluster_test

import (
	"testing"

	. "github.com/onsi/ginkgo"

	ctrlmgr "sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/acharyasreej/vm-operator/controllers/infracluster"
	ctrlContext "github.com/acharyasreej/vm-operator/pkg/context"
	providerfake "github.com/acharyasreej/vm-operator/pkg/vmprovider/fake"
	"github.com/acharyasreej/vm-operator/test/builder"
)

var intgFakeVMProvider = providerfake.NewVMProvider()

var suite = builder.NewTestSuiteForController(
	infracluster.AddToManager,
	func(ctx *ctrlContext.ControllerManagerContext, _ ctrlmgr.Manager) error {
		ctx.VMProvider = intgFakeVMProvider
		return nil
	},
)

var unitTests = func() {
	Describe("WCP ConfigMap", unitTestsWcpConfig)
}

func TestInfraclusterController(t *testing.T) {
	suite.Register(t, "Infracluster controller suite", intgTests, unitTests)
}

var _ = BeforeSuite(suite.BeforeSuite)

var _ = AfterSuite(suite.AfterSuite)
