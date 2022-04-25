// Copyright (c) 2019 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package mutation_test

import (
	"testing"

	. "github.com/onsi/ginkgo"

	"github.com/acharyasreej/vm-operator/test/builder"
	"github.com/acharyasreej/vm-operator/webhooks/virtualmachine/mutation"
)

// suite is used for unit and integration testing this webhook.
var suite = builder.NewTestSuiteForMutatingWebhook(
	mutation.AddToManager,
	mutation.NewMutator,
	"default.mutating.virtualmachine.vmoperator.vmware.com")

func TestWebhook(t *testing.T) {
	suite.Register(t, "Mutating webhook suite", intgTests, uniTests)
}

var _ = BeforeSuite(suite.BeforeSuite)

var _ = AfterSuite(suite.AfterSuite)
