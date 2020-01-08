/* **********************************************************
 * Copyright 2019 VMware, Inc.  All rights reserved. -- VMware Confidential
 * **********************************************************/

package integration

import (
	"context"
	"crypto/tls"
	"os"
	"strconv"

	"k8s.io/klog/klogr"

	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/simulator"

	_ "github.com/vmware/govmomi/vapi/simulator"
	_ "github.com/vmware-tanzu/vm-operator/pkg/vmprovider/providers/vsphere/cluster/simulator"
)

type VcSimInstance struct {
	VcSim  *simulator.Model
	Server *simulator.Server
	IP     string
	Port   int
}

var log = klogr.New()

func NewVcSimInstance() *VcSimInstance {
	vpx := simulator.VPX()
	err := vpx.Create()
	if err != nil {
		vpx.Remove()
		log.Error(err, "Fail to create vc simulator")
		os.Exit(255)
	}
	// Register imported simulators above (vapi/simulator, cluster/simulator)
	vpx.Service.RegisterEndpoints = true

	return &VcSimInstance{VcSim: vpx}
}

func (v *VcSimInstance) Start() (vcAddress string, vcPort int) {
	var err error
	v.VcSim.Service.TLS = new(tls.Config)
	v.Server = v.VcSim.Service.NewServer()
	v.IP = v.Server.URL.Hostname()
	v.Port, err = strconv.Atoi(v.Server.URL.Port())
	if err != nil {
		v.Server.Close()
		log.Error(err, "Fail to find vc simulator port")
		os.Exit(255)
	}
	//register for vapi/rest calls

	return v.IP, v.Port
}

func (v *VcSimInstance) Stop() {
	if v.Server != nil {
		v.Server.Close()
	}
	if v.VcSim != nil {
		v.VcSim.Remove()
	}
}

func (v *VcSimInstance) NewClient(ctx context.Context) (*govmomi.Client, error) {
	return govmomi.NewClient(ctx, v.Server.URL, true)
}
