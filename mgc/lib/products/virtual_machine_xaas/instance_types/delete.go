/*
Executor: delete

# Summary

# Delete Instance Type V1

# Description

Remove a instance type from virtual machine DB.

### Note
This route only remove the instance type from virtual machine DB.

Needs to remove the flavor from URP after use this route.

Version: 1.230.0

import "magalu.cloud/lib/products/virtual_machine_xaas/instance_types"
*/
package instanceTypes

import (
	mgcCore "magalu.cloud/core"
	mgcHelpers "magalu.cloud/lib/helpers"
)

type DeleteParameters struct {
	Id string `json:"id"`
}

type DeleteConfigs struct {
	Env       *string `json:"env,omitempty"`
	Region    *string `json:"region,omitempty"`
	ServerUrl *string `json:"serverUrl,omitempty"`
}

func (s *service) Delete(
	parameters DeleteParameters,
	configs DeleteConfigs,
) (
	err error,
) {
	exec, ctx, err := mgcHelpers.PrepareExecutor("Delete", mgcCore.RefPath("/virtual-machine-xaas/instance types/delete"), s.client, s.ctx)
	if err != nil {
		return
	}

	var p mgcCore.Parameters
	if p, err = mgcHelpers.ConvertParameters[DeleteParameters](parameters); err != nil {
		return
	}

	var c mgcCore.Configs
	if c, err = mgcHelpers.ConvertConfigs[DeleteConfigs](configs); err != nil {
		return
	}

	_, err = exec.Execute(ctx, p, c)
	return
}

func (s *service) DeleteConfirmPrompt(
	parameters DeleteParameters,
	configs DeleteConfigs,
) (message string) {
	e, err := mgcHelpers.ResolveExecutor("Delete", mgcCore.RefPath("/virtual-machine-xaas/instance types/delete"), s.client)
	if err != nil {
		return
	}

	exec, ok := e.(mgcCore.ConfirmableExecutor)
	if !ok {
		// Not expected, but let's return an empty message
		return
	}

	var p mgcCore.Parameters
	if p, err = mgcHelpers.ConvertParameters[DeleteParameters](parameters); err != nil {
		return
	}

	var c mgcCore.Configs
	if c, err = mgcHelpers.ConvertConfigs[DeleteConfigs](configs); err != nil {
		return
	}

	return exec.ConfirmPrompt(p, c)
}

// TODO: links
// TODO: related