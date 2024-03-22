/*
Executor: get

# Summary

# Retrieve the details of a snapshot

# Description

Retrieve details of a Snapshot for the currently authenticated tenant.

#### Notes
  - Use the **expand** argument to obtain additional details about the Volume
    used to create the Snapshot.
  - Utilize the **block-storage snapshots list** command to retrieve a list of
    all Snapshots and obtain the ID of the Snapshot for which you want to retrieve
    details.

Version: v1

import "magalu.cloud/lib/products/block_storage/snapshots"
*/
package snapshots

import (
	"context"

	mgcCore "magalu.cloud/core"
	mgcClient "magalu.cloud/lib"
	mgcHelpers "magalu.cloud/lib/helpers"
)

type GetParameters struct {
	Expand GetParametersExpand `json:"expand,omitempty"`
	Id     string              `json:"id"`
}

type GetParametersExpand []string

type GetConfigs struct {
	Env       string `json:"env,omitempty"`
	Region    string `json:"region,omitempty"`
	ServerUrl string `json:"serverUrl,omitempty"`
}

type GetResult struct {
	CreatedAt   string          `json:"created_at"`
	Description *string         `json:"description"`
	Id          string          `json:"id"`
	Name        string          `json:"name"`
	Size        int             `json:"size"`
	State       string          `json:"state"`
	Status      string          `json:"status"`
	UpdatedAt   string          `json:"updated_at"`
	Volume      GetResultVolume `json:"volume"`
}

// any of: GetResultVolume0, GetResultVolume1
type GetResultVolume struct {
	GetResultVolume0 `json:",squash"` // nolint
	GetResultVolume1 `json:",squash"` // nolint
}

type GetResultVolume0 struct {
	Id string `json:"id"`
}

type GetResultVolume1 struct {
	Id   string               `json:"id"`
	Name string               `json:"name"`
	Size int                  `json:"size"`
	Type GetResultVolume1Type `json:"type"`
}

type GetResultVolume1Type struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

func Get(
	client *mgcClient.Client,
	ctx context.Context,
	parameters GetParameters,
	configs GetConfigs,
) (
	result GetResult,
	err error,
) {
	exec, ctx, err := mgcHelpers.PrepareExecutor("Get", mgcCore.RefPath("/block-storage/snapshots/get"), client, ctx)
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

func GetUntilTermination(
	client *mgcClient.Client,
	ctx context.Context,
	parameters GetParameters,
	configs GetConfigs,
) (
	result GetResult,
	err error,
) {
	e, ctx, err := mgcHelpers.PrepareExecutor("Get", mgcCore.RefPath("/block-storage/snapshots/get"), client, ctx)
	if err != nil {
		return
	}

	exec, ok := e.(mgcCore.TerminatorExecutor)
	if !ok {
		// Not expected, but let's fallback
		return Get(
			client,
			ctx,
			parameters,
			configs,
		)
	}

	var p mgcCore.Parameters
	if p, err = mgcHelpers.ConvertParameters[GetParameters](parameters); err != nil {
		return
	}

	var c mgcCore.Configs
	if c, err = mgcHelpers.ConvertConfigs[GetConfigs](configs); err != nil {
		return
	}

	r, err := exec.ExecuteUntilTermination(ctx, p, c)
	if err != nil {
		return
	}
	return mgcHelpers.ConvertResult[GetResult](r)
}

// TODO: links
// TODO: related