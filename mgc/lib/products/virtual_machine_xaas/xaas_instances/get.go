/*
Executor: get

# Summary

# Retrieve the details of an instance in a xaas project

# Description

Get an instance details for the current tenant which is logged in.

#### Notes
- You can use the virtual-machine list command to retrieve all instances,
so you can get the id of the instance that you want to get details.

- You can use the **expand** argument to get more details from the inner objects
like image or type.

Version: 1.230.0

import "magalu.cloud/lib/products/virtual_machine_xaas/xaas_instances"
*/
package xaasInstances

import (
	mgcCore "magalu.cloud/core"
	mgcHelpers "magalu.cloud/lib/helpers"
)

type GetParameters struct {
	Expand      *GetParametersExpand `json:"expand,omitempty"`
	Id          string               `json:"id"`
	ProjectType string               `json:"project_type"`
}

type GetParametersExpand []string

type GetConfigs struct {
	Env       *string `json:"env,omitempty"`
	Region    *string `json:"region,omitempty"`
	ServerUrl *string `json:"serverUrl,omitempty"`
}

type GetResult struct {
	AvailabilityZone *string              `json:"availability_zone,omitempty"`
	CreatedAt        string               `json:"created_at"`
	Error            *GetResultError      `json:"error,omitempty"`
	Id               string               `json:"id"`
	Image            GetResultImage       `json:"image"`
	MachineType      GetResultMachineType `json:"machine_type"`
	Name             *string              `json:"name,omitempty"`
	Network          *GetResultNetwork    `json:"network,omitempty"`
	SshKeyName       *string              `json:"ssh_key_name,omitempty"`
	State            string               `json:"state"`
	Status           string               `json:"status"`
	UpdatedAt        *string              `json:"updated_at,omitempty"`
	UserData         *string              `json:"user_data,omitempty"`
}

type GetResultError struct {
	Message string `json:"message"`
	Slug    string `json:"slug"`
}

// any of: GetResultImage0, GetResultImage1
type GetResultImage struct {
	GetResultImage0 `json:",squash"` // nolint
	GetResultImage1 `json:",squash"` // nolint
}

type GetResultImage0 struct {
	Id string `json:"id"`
}

type GetResultImage1 struct {
	Id       string  `json:"id"`
	Name     string  `json:"name"`
	Platform *string `json:"platform,omitempty"`
}

// any of: GetResultMachineType0, GetResultMachineType1
type GetResultMachineType struct {
	GetResultMachineType0 `json:",squash"` // nolint
	GetResultMachineType1 `json:",squash"` // nolint
}

type GetResultMachineType0 struct {
	Id string `json:"id"`
}

type GetResultMachineType1 struct {
	Disk  int    `json:"disk"`
	Id    string `json:"id"`
	Name  string `json:"name"`
	Ram   int    `json:"ram"`
	Vcpus int    `json:"vcpus"`
}

// any of: GetResultNetwork0, GetResultNetwork1
type GetResultNetwork struct {
	GetResultNetwork0 `json:",squash"` // nolint
	GetResultNetwork1 `json:",squash"` // nolint
}

type GetResultNetwork0 struct {
	Ports GetResultNetwork0Ports `json:"ports"`
}

type GetResultNetwork0PortsItem struct {
	Id string `json:"id"`
}

type GetResultNetwork0Ports []GetResultNetwork0PortsItem

type GetResultNetwork1 struct {
	Ports *GetResultNetwork1Ports `json:"ports,omitempty"`
	Vpc   *GetResultNetwork1Vpc   `json:"vpc,omitempty"`
}

type GetResultNetwork1PortsItem struct {
	Id          string                                `json:"id"`
	IpAddresses GetResultNetwork1PortsItemIpAddresses `json:"ipAddresses"`
	Name        string                                `json:"name"`
}

type GetResultNetwork1PortsItemIpAddresses struct {
	IpV6address      *string `json:"ipV6Address,omitempty"`
	PrivateIpAddress string  `json:"privateIpAddress"`
	PublicIpAddress  *string `json:"publicIpAddress,omitempty"`
}

type GetResultNetwork1Ports []GetResultNetwork1PortsItem

type GetResultNetwork1Vpc struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

func (s *service) Get(
	parameters GetParameters,
	configs GetConfigs,
) (
	result GetResult,
	err error,
) {
	exec, ctx, err := mgcHelpers.PrepareExecutor("Get", mgcCore.RefPath("/virtual-machine-xaas/xaas instances/get"), s.client, s.ctx)
	if err != nil {
		return
	}

	var p mgcCore.Parameters
	if p, err = mgcHelpers.ConvertParameters[GetParameters](parameters); err != nil {
		return
	}

	var c mgcCore.Configs
	if c, err = mgcHelpers.ConvertConfigs[GetConfigs](configs); err != nil {
		return
	}

	r, err := exec.Execute(ctx, p, c)
	if err != nil {
		return
	}
	return mgcHelpers.ConvertResult[GetResult](r)
}

// TODO: links
// TODO: related