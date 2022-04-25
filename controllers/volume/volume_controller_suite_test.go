// Copyright (c) 2020 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package volume_test

import (
	"testing"

	. "github.com/onsi/ginkgo"

	"github.com/acharyasreej/vm-operator/controllers/volume"
	pkgmgr "github.com/acharyasreej/vm-operator/pkg/manager"
	"github.com/acharyasreej/vm-operator/test/builder"
)

var suite = builder.NewTestSuiteForController(
	volume.AddToManager,
	pkgmgr.InitializeProvidersNoopFn,
)

func TestVolume(t *testing.T) {
	suite.Register(t, "Volume controller suite", intgTests, unitTests)
}

var _ = BeforeSuite(suite.BeforeSuite)

var _ = AfterSuite(suite.AfterSuite)
