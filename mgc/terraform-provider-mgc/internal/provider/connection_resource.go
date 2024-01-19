package provider

import (
	"context"
	"fmt"
	"slices"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"magalu.cloud/core"
	mgcSdk "magalu.cloud/sdk"
)

const (
	createResultKey = "create-result"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &MgcConnectionResource{}
var _ resource.ResourceWithImportState = &MgcConnectionResource{}

// MgcConnectionResource defines Connection Resources via Links. Conenction Resources aren't real resources
// themselves, they represent conenctions to be taken regarding another resource. For example, turning
// a resource instance on on off, modifying its status, etc.
type MgcConnectionResource struct {
	sdk         *mgcSdk.Sdk
	name        tfName
	description string
	create      mgcSdk.Executor
	read        mgcSdk.Linker
	update      mgcSdk.Linker // TODO: Will conenction resources need/have updates?
	delete      mgcSdk.Linker
	inputAttr   resAttrInfoMap
	outputAttr  resAttrInfoMap
	tfschema    *schema.Schema
}

func newMgcConnectionResource(
	ctx context.Context,
	sdk *mgcSdk.Sdk,
	name tfName,
	description string,
	connection mgcSdk.Executor,
	sourceDelete mgcSdk.Executor,
) (*MgcConnectionResource, error) {
	var read, update, delete mgcSdk.Linker
	for k, link := range connection.Links() {
		switch k {
		case "get-connection":
			read = link
		case "update-connection":
			update = link
		case "delete-connection":
			delete = link
		}
	}

	if read == nil {
		return nil, fmt.Errorf("Connection Resource %q misses read", name)
	}
	if delete == nil {
		return nil, fmt.Errorf("Connection Resource %q misses delete", name)
	}
	if delete.ResultSchema() == sourceDelete.ResultSchema() {
		return nil, fmt.Errorf("Connection Resource %q's delete link targets the source resource deletion, not the connection deletion", name)
	}
	if update == nil {
		tflog.Warn(ctx, fmt.Sprintf("Connection Resource %s misses update operations", name))
		update = core.NewSimpleLink(core.SimpleLinkSpec{
			Owner:  connection,
			Target: core.NoOpExecutor(),
		})
	}
	return &MgcConnectionResource{
		sdk:         sdk,
		name:        name,
		description: description,
		create:      connection,
		read:        read,
		update:      update,
		delete:      delete,
	}, nil
}

func (r *MgcConnectionResource) getReadParamsModifiers(ctx context.Context, mgcSchema *mgcSdk.Schema, mgcName mgcName) attributeModifiers {
	isRequired := slices.Contains(mgcSchema.Required, string(mgcName))
	return attributeModifiers{
		isRequired:                 isRequired,
		isOptional:                 !isRequired,
		isComputed:                 false,
		useStateForUnknown:         true,
		requiresReplaceWhenChanged: true,
		getChildModifiers:          getInputChildModifiers,
	}
}

func (r *MgcConnectionResource) getDeleteParamsModifiers(ctx context.Context, mgcSchema *mgcSdk.Schema, mgcName mgcName) attributeModifiers {
	// TODO: For now we consider all delete params as optionals, we need to think a way for the user to define
	// required delete params
	return attributeModifiers{
		isRequired:                 false,
		isOptional:                 true,
		isComputed:                 false,
		useStateForUnknown:         true,
		requiresReplaceWhenChanged: true,
		getChildModifiers:          getInputChildModifiers,
	}
}

func (r *MgcConnectionResource) InputAttrInfoMap(ctx context.Context, d *Diagnostics) resAttrInfoMap {
	if r.inputAttr == nil {
		r.inputAttr = generateResAttrInfoMap(ctx, r.name,
			[]resAttrInfoGenMetadata{
				{r.create.ParametersSchema(), r.getReadParamsModifiers},
				{r.read.AdditionalParametersSchema(), r.getReadParamsModifiers},
				{r.delete.AdditionalParametersSchema(), r.getDeleteParamsModifiers},
			}, d,
		)
	}
	return r.inputAttr
}

func (r *MgcConnectionResource) OutputAttrInfoMap(ctx context.Context, d *Diagnostics) resAttrInfoMap {
	if r.outputAttr == nil {
		r.outputAttr = generateResAttrInfoMap(ctx, r.name,
			[]resAttrInfoGenMetadata{
				{r.create.ResultSchema(), getResultModifiers},
				{r.read.ResultSchema(), getResultModifiers},
			}, d,
		)
	}
	return r.outputAttr
}

func (r *MgcConnectionResource) attrTree(ctx context.Context) (tree resAttrInfoTree, d Diagnostics) {
	return resAttrInfoTree{input: r.InputAttrInfoMap(ctx, &d), output: r.OutputAttrInfoMap(ctx, &d)}, d
}

// BEGIN: Resource implementation

func (r *MgcConnectionResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = string(r.name)
}

func (r *MgcConnectionResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	if r.tfschema == nil {
		ctx = tflog.SetField(ctx, resourceNameField, r.name)
		attrTree, d := r.attrTree(ctx)
		resp.Diagnostics.Append(d...)
		if resp.Diagnostics.HasError() {
			return
		}

		tfs := generateTFSchema(ctx, r.name, r.description, attrTree, (*Diagnostics)(&resp.Diagnostics))
		r.tfschema = &tfs
	}
	resp.Schema = *r.tfschema
}

func (r *MgcConnectionResource) newLinkOperation(
	link core.Linker,
	attrTree resAttrInfoTree,
	getPrivateStateKey func(context.Context, string) ([]byte, diag.Diagnostics),
	setPrivateStateKey func(context.Context, string, []byte) diag.Diagnostics,
	constructor func(tfName, resAttrInfoTree, core.Executor) MgcOperation,
) MgcOperation {
	return newMgcConnectionLink(r.name, attrTree, getPrivateStateKey, setPrivateStateKey, r.create, func(result core.Result) MgcOperation {
		exec, err := link.CreateExecutor(result)
		if err != nil {
			return nil
		}
		return newMgcConnectionRead(r.name, attrTree, exec)
	})
}

func (r *MgcConnectionResource) newReadOperation(
	attrTree resAttrInfoTree,
	getPrivateStateKey func(context.Context, string) ([]byte, diag.Diagnostics),
	setPrivateStateKey func(context.Context, string, []byte) diag.Diagnostics,
) MgcReadOperation {
	readOp := r.newLinkOperation(r.read, attrTree, getPrivateStateKey, setPrivateStateKey, newMgcConnectionRead)
	return (wrapReadOperation(readOp, r.read.ResultSchema()))
}

func (r *MgcConnectionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	diagnostics := Diagnostics{}
	defer func() {
		resp.Diagnostics = diag.Diagnostics(diagnostics)
	}()

	attrTree, d := r.attrTree(ctx)
	if diagnostics.AppendCheckError(d...) {
		return
	}

	createOp := newMgcConnectionCreate(r.name, attrTree, resp.Private.GetKey, resp.Private.SetKey, r.create, r.delete)
	readOp := r.newReadOperation(attrTree, resp.Private.GetKey, resp.Private.SetKey)
	runner := newMgcOperationRunner(r.sdk, createOp, readOp, tfsdk.State(req.Plan), req.Plan, &resp.State)
	d = runner.Run(ctx)
	diagnostics.Append(d...)
}

func (r *MgcConnectionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	diagnostics := Diagnostics{}
	defer func() {
		resp.Diagnostics = diag.Diagnostics(diagnostics)
	}()

	attrTree, d := r.attrTree(ctx)
	if diagnostics.AppendCheckError(d...) {
		return
	}

	readOp := r.newReadOperation(attrTree, req.Private.GetKey, resp.Private.SetKey)
	runner := newMgcOperationRunner(r.sdk, readOp, readOp, req.State, tfsdk.Plan(req.State), &resp.State)
	d = runner.Run(ctx)
	diagnostics.Append(d...)
}

// Update will most likely never be called, as we always require replace when changed
func (r *MgcConnectionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// no-op
}

func (r *MgcConnectionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	diagnostics := Diagnostics{}
	defer func() {
		resp.Diagnostics = diag.Diagnostics(diagnostics)
	}()

	attrTree, d := r.attrTree(ctx)
	if diagnostics.AppendCheckError(d...) {
		return
	}

	deleteOp := r.newLinkOperation(r.delete, attrTree, req.Private.GetKey, req.Private.SetKey, newMgcConnectionDelete)
	runner := newMgcOperationRunner(r.sdk, deleteOp, nil, req.State, tfsdk.Plan(req.State), &resp.State)
	d = runner.Run(ctx)
	diagnostics.Append(d...)
}

func (r *MgcConnectionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

var _ resource.Resource = (*MgcConnectionResource)(nil)

// END: Resource implementation
