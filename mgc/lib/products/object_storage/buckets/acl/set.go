/*
Executor: set

# Description

set permission information for the specified bucket

import "magalu.cloud/lib/products/object_storage/buckets/acl"
*/
package acl

import (
	"context"

	mgcCore "magalu.cloud/core"
	mgcClient "magalu.cloud/lib"
	mgcHelpers "magalu.cloud/lib/helpers"
)

type SetParameters struct {
	AuthenticatedRead bool                          `json:"authenticated_read,omitempty"`
	Bucket            string                        `json:"bucket"`
	GrantFullControl  SetParametersGrantFullControl `json:"grant_full_control,omitempty"`
	GrantRead         SetParametersGrantRead        `json:"grant_read,omitempty"`
	GrantReadAcp      SetParametersGrantReadAcp     `json:"grant_read_acp,omitempty"`
	GrantWrite        SetParametersGrantWrite       `json:"grant_write,omitempty"`
	GrantWriteAcp     SetParametersGrantWriteAcp    `json:"grant_write_acp,omitempty"`
	Private           bool                          `json:"private,omitempty"`
	PublicRead        bool                          `json:"public_read,omitempty"`
	PublicReadWrite   bool                          `json:"public_read_write,omitempty"`
}

type SetParametersGrantFullControlItem struct {
	Id string `json:"id"`
}

type SetParametersGrantFullControl []SetParametersGrantFullControlItem

type SetParametersGrantRead []SetParametersGrantFullControlItem

type SetParametersGrantReadAcp []SetParametersGrantFullControlItem

type SetParametersGrantWrite []SetParametersGrantFullControlItem

type SetParametersGrantWriteAcp []SetParametersGrantFullControlItem

type SetConfigs struct {
	ChunkSize int    `json:"chunkSize,omitempty"`
	Env       string `json:"env,omitempty"`
	Region    string `json:"region,omitempty"`
	ServerUrl string `json:"serverUrl,omitempty"`
	Workers   int    `json:"workers,omitempty"`
}

type SetResult any

func Set(
	client *mgcClient.Client,
	ctx context.Context,
	parameters SetParameters,
	configs SetConfigs,
) (
	result SetResult,
	err error,
) {
	exec, ctx, err := mgcHelpers.PrepareExecutor("Set", mgcCore.RefPath("/object-storage/buckets/acl/set"), client, ctx)
	if err != nil {
		return
	}

	var p mgcCore.Parameters
	if p, err = mgcHelpers.ConvertParameters[SetParameters](parameters); err != nil {
		return
	}

	var c mgcCore.Configs
	if c, err = mgcHelpers.ConvertConfigs[SetConfigs](configs); err != nil {
		return
	}

	r, err := exec.Execute(ctx, p, c)
	if err != nil {
		return
	}
	return mgcHelpers.ConvertResult[SetResult](r)
}

// TODO: links
// TODO: related