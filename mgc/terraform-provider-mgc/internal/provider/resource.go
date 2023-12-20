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
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"magalu.cloud/core"
	mgcSchemaPkg "magalu.cloud/core/schema"
	"magalu.cloud/core/utils"
	mgcSdk "magalu.cloud/sdk"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &MgcResource{}
var _ resource.ResourceWithImportState = &MgcResource{}

// MgcResource defines the resource implementation.
type MgcResource struct {
	sdk               *mgcSdk.Sdk
	name              string
	description       string
	create            mgcSdk.Executor
	read              mgcSdk.Executor
	update            mgcSdk.Executor
	delete            mgcSdk.Executor
	inputAttrInfoMap  resAttrInfoMap
	outputAttrInfoMap resAttrInfoMap
	splitAttributes   []splitResAttribute
	tfschema          *schema.Schema
	propertySetters   map[mgcName]propertySetter
}

func newMgcResource(
	ctx context.Context,
	sdk *mgcSdk.Sdk,
	name, description string,
	create, read, update, delete, list mgcSdk.Executor,
) (*MgcResource, error) {
	if create == nil {
		return nil, fmt.Errorf("resource %q misses create", name)
	}
	if delete == nil {
		return nil, fmt.Errorf("resource %q misses delete", name)
	}
	if read == nil {
		if list == nil {
			return nil, fmt.Errorf("resource %q misses read", name)
		}

		readFromList, err := createReadFromList(list, create.ResultSchema(), delete.ParametersSchema())
		if err != nil {
			tflog.Warn(ctx, fmt.Sprintf("unable to generate 'read' operation from 'list' for %q", name), map[string]any{"error": err})
			return nil, fmt.Errorf("resource %q misses read", name)
		}

		read = readFromList
		tflog.Debug(ctx, fmt.Sprintf("generated 'read' operation based on 'list' for %q", name))
	}
	if update == nil {
		update = core.NoOpExecutor()
	}

	propertySetterContainer, err := collectPropertySetterContainers(read.Links())
	if err != nil {
		return nil, err
	}

	var propertySetters map[mgcName]propertySetter
	for key, container := range propertySetterContainer {
		if propertySetters == nil {
			propertySetters = make(map[mgcName]propertySetter, len(propertySetterContainer))
		}

		switch container.argCount {
		case 0:
			propertySetters[key] = newDefaultPropertySetter(container.entries[0].target)
		case 1:
			propertySetters[key] = newEnumPropertySetter(container.entries)
		case 2:
			propertySetters[key] = newStrTransitionPropertySetter(container.entries)
		default:
			// TODO: Handle more action types?
			continue
		}
	}

	return &MgcResource{
		sdk:             sdk,
		name:            name,
		description:     description,
		create:          create,
		read:            read,
		update:          update,
		delete:          delete,
		propertySetters: propertySetters,
	}, nil
}

func (r *MgcResource) doesPropHaveSetter(name mgcName) bool {
	_, ok := r.propertySetters[name]
	return ok
}

// BEGIN: tfSchemaHandler implementation

func (r *MgcResource) Name() string {
	return r.name
}

func (r *MgcResource) Description() string {
	return r.description
}

func (r *MgcResource) getCreateParamsModifiers(ctx context.Context, mgcSchema *mgcSdk.Schema, mgcName mgcName) attributeModifiers {
	k := string(mgcName)
	isRequired := slices.Contains(mgcSchema.Required, k)
	isComputed := !isRequired
	if isComputed {
		readSchema := r.read.ResultSchema().Properties[k]
		if readSchema == nil {
			isComputed = false
		} else {
			// If not required and present in read it can be compute
			isComputed = mgcSchemaPkg.CheckSimilarJsonSchemas((*core.Schema)(readSchema.Value), (*core.Schema)(mgcSchema.Properties[string(mgcName)].Value))
		}
	}

	return attributeModifiers{
		isRequired:                 isRequired,
		isOptional:                 !isRequired,
		isComputed:                 isComputed,
		useStateForUnknown:         false,
		requiresReplaceWhenChanged: r.update.ParametersSchema().Properties[k] == nil && !r.doesPropHaveSetter(mgcName),
		getChildModifiers:          getInputChildModifiers,
	}
}

func (r *MgcResource) getUpdateParamsModifiers(ctx context.Context, mgcSchema *mgcSdk.Schema, mgcName mgcName) attributeModifiers {
	k := string(mgcName)
	isComputed := r.create.ResultSchema().Properties[k] != nil
	required := slices.Contains(mgcSchema.Required, k)

	return attributeModifiers{
		isRequired:                 required && !isComputed,
		isOptional:                 !required && !isComputed,
		isComputed:                 !required || isComputed,
		useStateForUnknown:         true,
		requiresReplaceWhenChanged: false,
		getChildModifiers:          getInputChildModifiers,
	}
}

func (r *MgcResource) getDeleteParamsModifiers(ctx context.Context, mgcSchema *mgcSdk.Schema, mgcName mgcName) attributeModifiers {
	isComputed := r.create.ResultSchema().Properties[string(mgcName)] != nil

	// All Delete parameters need to be optional, since they're not returned by the server when creating the resources,
	// and thus would produce "inconsistent results" in Terraform. Since we calculate the 'Update' parameters first,
	// all strictly necessary parameters for deletion (like the resource ID) will already be computed correctly (in the Update
	// parameters)
	return attributeModifiers{
		isRequired:                 false,
		isOptional:                 true,
		isComputed:                 isComputed,
		useStateForUnknown:         true,
		requiresReplaceWhenChanged: false,
		getChildModifiers:          getInputChildModifiers,
	}
}

func (r *MgcResource) getResultModifiers(ctx context.Context, mgcSchema *mgcSdk.Schema, mgcName mgcName) attributeModifiers {
	return attributeModifiers{
		isRequired:                 false,
		isOptional:                 r.doesPropHaveSetter(mgcName),
		isComputed:                 true,
		useStateForUnknown:         false,
		requiresReplaceWhenChanged: false,
		getChildModifiers:          getResultModifiers,
	}
}

func (r *MgcResource) AppendSplitAttribute(split splitResAttribute) {
	if r.splitAttributes == nil {
		r.splitAttributes = []splitResAttribute{}
	}
	r.splitAttributes = append(r.splitAttributes, split)
}

var _ tfSchemaHandler = (*MgcResource)(nil)

// END: tfSchemaHandler implementation

// BEGIN: tfStateHandler implementation

func (r *MgcResource) InputAttrInfoMap(ctx context.Context, d *diag.Diagnostics) resAttrInfoMap {
	if r.inputAttrInfoMap == nil {
		r.inputAttrInfoMap = generateResAttrInfoMap(ctx, r.name,
			[]resAttrInfoGenMetadata{
				{r.create.ParametersSchema(), r.getCreateParamsModifiers},
				{r.update.ParametersSchema(), r.getUpdateParamsModifiers},
				{r.delete.ParametersSchema(), r.getDeleteParamsModifiers},
			}, d,
		)
	}
	return r.inputAttrInfoMap
}

func (r *MgcResource) OutputAttrInfoMap(ctx context.Context, d *diag.Diagnostics) resAttrInfoMap {
	if r.outputAttrInfoMap == nil {
		r.outputAttrInfoMap = generateResAttrInfoMap(ctx, r.name,
			[]resAttrInfoGenMetadata{
				{r.create.ResultSchema(), r.getResultModifiers},
				{r.read.ResultSchema(), r.getResultModifiers},
			}, d,
		)
	}
	return r.outputAttrInfoMap
}

func (r *MgcResource) SplitAttributes() []splitResAttribute {
	return r.splitAttributes
}

func (r *MgcResource) TFSchema() *schema.Schema {
	return r.tfschema
}

func (r *MgcResource) ReadResultSchema() *mgcSdk.Schema {
	return r.read.ResultSchema()
}

var _ tfStateHandler = (*MgcResource)(nil)

// END: tfStateHandler implementation

// BEGIN: Resource implementation

func (r *MgcResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = r.name
}

func (r *MgcResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	if r.tfschema == nil {
		ctx = tflog.SetField(ctx, resourceNameField, r.name)
		tfs := generateTFSchema(r, ctx, &resp.Diagnostics)
		r.tfschema = &tfs
	}
	resp.Schema = *r.tfschema
}

func (r *MgcResource) performOperation(
	ctx context.Context,
	exec core.Executor,
	inState tfsdk.State,
	d *diag.Diagnostics,
) core.ResultWithValue {
	configs := getConfigs(ctx, exec.ConfigsSchema())
	params := readMgcMapSchemaFromTFState(r, exec.ParametersSchema(), ctx, inState, d)
	if d.HasError() {
		return nil
	}

	return execute(r.name, ctx, exec, params, configs, d)
}

func (r *MgcResource) findAttrWithTFName(ctx context.Context, name tfName, d *diag.Diagnostics) (*resAttrInfo, bool) {
	for _, inputAttr := range r.InputAttrInfoMap(ctx, d) {
		if inputAttr.tfName == name {
			return inputAttr, true
		}
	}
	for _, outputAttr := range r.OutputAttrInfoMap(ctx, d) {
		if outputAttr.tfName == name {
			return outputAttr, true
		}
	}
	return nil, false
}

func (r *MgcResource) callPostOpPropertySetters(
	ctx context.Context,
	inState tfsdk.State,
	outState *tfsdk.State,
	readResult core.ResultWithValue,
	readResultMap map[string]any,
	d *diag.Diagnostics,
) {
	var curStateAsMap map[string]tftypes.Value
	err := inState.Raw.As(&curStateAsMap)
	if err != nil {
		tflog.Warn(
			ctx,
			fmt.Sprintf("[resource] unable to get current state as map: %v. Won't look for property setters", err),
			map[string]any{"resource": r.Name()},
		)
		return
	}

	tflog.Debug(
		ctx,
		"[resource] will look for property setters to perform",
		map[string]any{"resource": r.Name()},
	)
	for stateValKey, stateVal := range curStateAsMap {
		tflog.Debug(
			ctx,
			fmt.Sprintf("[resource] looking for property setter for %q", stateValKey),
			map[string]any{"resource": r.Name()},
		)

		var currentVal any
		for k, v := range readResultMap {
			if mgcName(k).asTFName() == tfName(stateValKey) {
				currentVal = v
				break
			}
		}
		if currentVal == nil {
			tflog.Debug(
				ctx,
				"[resource] unable to find matching key in read result",
				map[string]any{"resource": r.Name(), "read result": readResultMap},
			)
			continue
		}

		attr, ok := r.findAttrWithTFName(ctx, tfName(stateValKey), d)
		if !ok || d.HasError() {
			tflog.Debug(
				ctx,
				"[resource] unable to find attribute corresponding to TF prop key",
				map[string]any{"resource": r.Name(), "propTFName": stateValKey, "inputAttributes": r.InputAttrInfoMap(ctx, d), "outputAttr": r.OutputAttrInfoMap(ctx, d)},
			)
			continue
		}

		convDiag := diag.Diagnostics{}
		conv := newTFStateLoader(ctx, &convDiag)
		targetVal, ok := conv.loadMgcSchemaValue(attr, stateVal, false, false)
		if convDiag.HasError() || !ok {
			tflog.Debug(
				ctx,
				"[resource] unable to convert TF value to MGC value to update prop, ignoring set",
				map[string]any{"resource": r.Name(), "propTFName": attr.tfName, "tfValue": stateVal},
			)
			continue
		}

		if utils.IsSameValueOrPointer(currentVal, targetVal) {
			tflog.Debug(
				ctx,
				"[resource] no need, current state matches desired state",
				map[string]any{"resource": r.Name()},
			)
			continue
		}

		setter, ok := r.propertySetters[attr.mgcName]
		if !ok {
			tflog.Warn(
				ctx,
				"[resource] unable to find property setter to update prop, will result in error",
				map[string]any{"resource": r.Name(), "propTFName": attr.tfName, "propMGCName": attr.mgcName},
			)
			continue
		}

		r.callPropertySetter(ctx, inState, outState, attr.tfName, setter, currentVal, targetVal, readResult, d)
		if d.HasError() {
			return
		}
	}
}

func (r *MgcResource) callPropertySetter(
	ctx context.Context,
	inState tfsdk.State,
	outState *tfsdk.State,
	propTFName tfName,
	setter propertySetter,
	currentVal, targetVal core.Value,
	readResult core.ResultWithValue,
	d *diag.Diagnostics,
) {
	tflog.Debug(
		ctx,
		fmt.Sprintf("[resource] will update param %q with property setter", propTFName),
		map[string]any{"resource": r.Name()},
	)
	target, err := setter.getTarget(currentVal, targetVal)
	if err != nil {
		d.AddError(
			"property setter fail",
			fmt.Sprintf("unable to call property setter to update parameter %q: %v", propTFName, err),
		)
		return
	}

	setterExec, err := target.CreateExecutor(readResult)
	if err != nil {
		d.AddError(
			"property setter fail",
			fmt.Sprintf("unable to call property setter to update parameter %q: %v", propTFName, err),
		)
		return
	}

	propSetResult := r.performOperation(ctx, setterExec, inState, d)
	if d.HasError() {
		return
	}
	applyStateAfter(r, propSetResult, r.read, ctx, outState, d)
}

func (r *MgcResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	ctx = tflog.SetField(ctx, rpcField, "create")
	ctx = tflog.SetField(ctx, resourceNameField, r.name)
	ctx = r.sdk.WrapContext(ctx)

	createResult := r.performOperation(ctx, r.create, tfsdk.State(req.Plan), &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	readResult, readResultMap := applyStateAfter(r, createResult, r.read, ctx, &resp.State, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	r.callPostOpPropertySetters(ctx, tfsdk.State(req.Plan), &resp.State, readResult, readResultMap, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "resource created")
}

func (r *MgcResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	ctx = tflog.SetField(ctx, rpcField, "read")
	ctx = tflog.SetField(ctx, resourceNameField, r.name)
	ctx = r.sdk.WrapContext(ctx)

	readResult := r.performOperation(ctx, r.read, req.State, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	applyStateAfter(r, readResult, r.read, ctx, &resp.State, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "resource read")
}

func (r *MgcResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	ctx = tflog.SetField(ctx, rpcField, "update")
	ctx = tflog.SetField(ctx, resourceNameField, r.name)
	ctx = r.sdk.WrapContext(ctx)

	updateResult := r.performOperation(ctx, r.update, tfsdk.State(req.Plan), &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	readResult, readResultMap := applyStateAfter(r, updateResult, r.read, ctx, &resp.State, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	r.callPostOpPropertySetters(ctx, tfsdk.State(req.Plan), &resp.State, readResult, readResultMap, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "resource updated")
}

func (r *MgcResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	ctx = tflog.SetField(ctx, rpcField, "delete")
	ctx = tflog.SetField(ctx, resourceNameField, r.name)
	ctx = r.sdk.WrapContext(ctx)

	r.performOperation(ctx, r.delete, req.State, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "resource deleted")
}

func (r *MgcResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	ctx = tflog.SetField(ctx, rpcField, "import-state")
	ctx = tflog.SetField(ctx, resourceNameField, r.name)
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// END: Resource implemenation
