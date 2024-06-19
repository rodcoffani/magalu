/*
Executor: list

# Summary

Retrieves all machine-types available in the region.

# Description

Retrieves a list of machine types allowed for the current tenant which is logged in.

Version: 1.230.0

import "magalu.cloud/lib/products/virtual_machine_xaas/xaas_machine_types"
*/
package xaasMachineTypes

import (
	mgcCore "magalu.cloud/core"
	mgcHelpers "magalu.cloud/lib/helpers"
)

type ListParameters struct {
	Limit  *int    `json:"_limit,omitempty"`
	Offset *int    `json:"_offset,omitempty"`
	Sort   *string `json:"_sort,omitempty"`
}

type ListConfigs struct {
	Env       *string `json:"env,omitempty"`
	Region    *string `json:"region,omitempty"`
	ServerUrl *string `json:"serverUrl,omitempty"`
}

type ListResult struct {
	InstanceTypes ListResultInstanceTypes `json:"instance_types"`
}

// any of: ListResultInstanceTypesItem0, ListResultInstanceTypesItem1
type ListResultInstanceTypesItem struct {
	ListResultInstanceTypesItem0 `json:",squash"` // nolint
	ListResultInstanceTypesItem1 `json:",squash"` // nolint
}

type ListResultInstanceTypesItem0 struct {
	Disk   int    `json:"disk"`
	Gpu    *int   `json:"gpu,omitempty"`
	Id     string `json:"id"`
	Name   string `json:"name"`
	Ram    int    `json:"ram"`
	Sku    string `json:"sku"`
	Status string `json:"status"`
	Vcpus  int    `json:"vcpus"`
}

type ListResultInstanceTypesItem1 struct {
	Disk   int    `json:"disk"`
	Gpu    *int   `json:"gpu,omitempty"`
	Id     string `json:"id"`
	Name   string `json:"name"`
	Ram    int    `json:"ram"`
	Sku    string `json:"sku"`
	Status string `json:"status"`
	Vcpus  int    `json:"vcpus"`
}

type ListResultInstanceTypes []ListResultInstanceTypesItem

func (s *service) List(
	parameters ListParameters,
	configs ListConfigs,
) (
	result ListResult,
	err error,
) {
	exec, ctx, err := mgcHelpers.PrepareExecutor("List", mgcCore.RefPath("/virtual-machine-xaas/xaas machine-types/list"), s.client, s.ctx)
	if err != nil {
		return
	}

	var p mgcCore.Parameters
	if p, err = mgcHelpers.ConvertParameters[ListParameters](parameters); err != nil {
		return
	}

	var c mgcCore.Configs
	if c, err = mgcHelpers.ConvertConfigs[ListConfigs](configs); err != nil {
		return
	}

	r, err := exec.Execute(ctx, p, c)
	if err != nil {
		return
	}
	return mgcHelpers.ConvertResult[ListResult](r)
}

// TODO: links
// TODO: related