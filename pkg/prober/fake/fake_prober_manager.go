// Copyright (c) 2020-2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package fake

import (
	"context"
	"sync"

	vmoperatorv1alpha1 "github.com/acharyasreej/vm-operator-api/api/v1alpha1"

	"github.com/acharyasreej/vm-operator/pkg/prober"
)

type funcs struct {
	AddToProberManagerFn      func(vm *vmoperatorv1alpha1.VirtualMachine)
	RemoveFromProberManagerFn func(vm *vmoperatorv1alpha1.VirtualMachine)
}

type ProberManager struct {
	funcs
	sync.Mutex
	IsAddToProberManagerCalled      bool
	IsRemoveFromProberManagerCalled bool
}

func NewFakeProberManager() prober.Manager {
	return &ProberManager{}
}

func (m *ProberManager) Start(ctx context.Context) error {
	<-ctx.Done()

	return nil
}

func (m *ProberManager) AddToProberManager(vm *vmoperatorv1alpha1.VirtualMachine) {
	m.Lock()
	defer m.Unlock()

	if m.AddToProberManagerFn != nil {
		m.AddToProberManagerFn(vm)
		return
	}
}

func (m *ProberManager) RemoveFromProberManager(vm *vmoperatorv1alpha1.VirtualMachine) {
	m.Lock()
	defer m.Unlock()

	if m.RemoveFromProberManagerFn != nil {
		m.RemoveFromProberManagerFn(vm)
		return
	}
}
