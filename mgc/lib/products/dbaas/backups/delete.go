/*
Executor: delete

# Summary

Delete Backup.

# Description

Deletes a database backup.

Version: 1.15.3

import "magalu.cloud/lib/products/dbaas/backups"
*/
package backups

import (
	"context"

	mgcCore "magalu.cloud/core"
	mgcClient "magalu.cloud/lib"
	mgcHelpers "magalu.cloud/lib/helpers"
)

type DeleteParameters struct {
	BackupId string `json:"backup_id"`
	Exchange string `json:"exchange,omitempty"`
}

type DeleteConfigs struct {
	Env       string `json:"env,omitempty"`
	Region    string `json:"region,omitempty"`
	ServerUrl string `json:"serverUrl,omitempty"`
}

type DeleteResult any

func Delete(
	client *mgcClient.Client,
	ctx context.Context,
	parameters DeleteParameters,
	configs DeleteConfigs,
) (
	result DeleteResult,
	err error,
) {
	exec, ctx, err := mgcHelpers.PrepareExecutor("Delete", mgcCore.RefPath("/dbaas/backups/delete"), client, ctx)
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

	r, err := exec.Execute(ctx, p, c)
	if err != nil {
		return
	}
	return mgcHelpers.ConvertResult[DeleteResult](r)
}

func DeleteConfirmPrompt(
	client *mgcClient.Client,
	parameters DeleteParameters,
	configs DeleteConfigs,
) (message string) {
	e, err := mgcHelpers.ResolveExecutor("Delete", mgcCore.RefPath("/dbaas/backups/delete"), client)
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