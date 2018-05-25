package iso

import (
	"context"
	"fmt"

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
	Storage            []driver.DiskConfig `mapstructure:"storage"`
	Network            string              `mapstructure:"network"`
	NetworkCard        string              `mapstructure:"network_card"`
	USBController      bool                `mapstructure:"usb_controller"`
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
		Network:             s.Config.Network,
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
