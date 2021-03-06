// Copyright (c) 2019 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package virtualmachineservice

import (
	"github.com/pkg/errors"

	ctrlmgr "sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/acharyasreej/vm-operator/pkg/context"
	"github.com/acharyasreej/vm-operator/webhooks/virtualmachineservice/mutation"
	"github.com/acharyasreej/vm-operator/webhooks/virtualmachineservice/validation"
)

func AddToManager(ctx *context.ControllerManagerContext, mgr ctrlmgr.Manager) error {
	if err := validation.AddToManager(ctx, mgr); err != nil {
		return errors.Wrap(err, "failed to initialize validation webhook")
	}
	if err := mutation.AddToManager(ctx, mgr); err != nil {
		return errors.Wrap(err, "failed to initialize mutation webhook")
	}
	return nil
}
