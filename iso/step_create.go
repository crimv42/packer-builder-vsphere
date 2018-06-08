package iso

import (
	"context"
	"fmt"
	"net"
	"os"

	packerCommon "github.com/hashicorp/packer/common"
	"github.com/hashicorp/packer/helper/multistep"
	"github.com/hashicorp/packer/packer"
	"github.com/jetbrains-infra/packer-builder-vsphere/common"
	"github.com/jetbrains-infra/packer-builder-vsphere/driver"
)

type CreateConfig struct {
	Version     uint   `mapstructure:"vm_version"`
	GuestOSType string `mapstructure:"guest_os_type"`

	DiskControllerType string              `mapstructure:"disk_controller_type"`
	GlobalDiskType     string              `mapstructure:"disk_type"`
	HTTPIP             string              `mapstructure:"http_ip"`
	NetworkCard        string              `mapstructure:"network_card"`
	Networks           []string            `mapstructure:"networks"`
	Storage            []driver.DiskConfig `mapstructure:"storage"`
	USBController      bool                `mapstructure:"usb_controller"`
}

func getHostIP(s string) string {
	if net.ParseIP(s) != nil {
		return s
	}

	var ipaddr string
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		fmt.Println(err)
		os.Exit(2)
	}

	for _, a := range addrs {
		if ip, ok := a.(*net.IPNet); ok && !ip.IP.IsLoopback() {
			ipaddr = ip.IP.String()
			break
		}
	}
	return ipaddr
}

func (c *CreateConfig) Prepare() []error {
	var errs []error

	if c.GuestOSType == "" {
		c.GuestOSType = "otherGuest"
	}

	return errs
}

type StepCreateVM struct {
	Config   *CreateConfig
	Location *common.LocationConfig
}

func (s *StepCreateVM) Run(_ context.Context, state multistep.StateBag) multistep.StepAction {
	ui := state.Get("ui").(packer.Ui)
	d := state.Get("driver").(*driver.Driver)

	packerCommon.SetHTTPIP(getHostIP(s.Config.HTTPIP))

	ui.Say("Creating VM...")
	vm, err := d.CreateVM(&driver.CreateConfig{
		Cluster:             s.Location.Cluster,
		Datastore:           s.Location.Datastore,
		Folder:              s.Location.Folder,
		Host:                s.Location.Host,
		Name:                s.Location.VMName,
		ResourcePool:        s.Location.ResourcePool,
		DiskControllerType:  s.Config.DiskControllerType,
		GlobalDiskType:      s.Config.GlobalDiskType,
		GuestOS:             s.Config.GuestOSType,
		Networks:            s.Config.Networks,
		NetworkCard:         s.Config.NetworkCard,
		Storage:             s.Config.Storage,
		USBController:       s.Config.USBController,
		Version:             s.Config.Version,
	})
	if err != nil {
		state.Put("error", fmt.Errorf("error creating vm: %v", err))
		return multistep.ActionHalt
	}
	state.Put("vm", vm)

	return multistep.ActionContinue
}

func (s *StepCreateVM) Cleanup(state multistep.StateBag) {
	_, cancelled := state.GetOk(multistep.StateCancelled)
	_, halted := state.GetOk(multistep.StateHalted)
	if !cancelled && !halted {
		return
	}

	ui := state.Get("ui").(packer.Ui)

	st := state.Get("vm")
	if st == nil {
		return
	}
	vm := st.(*driver.VirtualMachine)

	ui.Say("Destroying VM...")
	err := vm.Destroy()
	if err != nil {
		ui.Error(err.Error())
	}
}
