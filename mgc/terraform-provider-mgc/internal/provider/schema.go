package provider

import (
	"context"
	"fmt"
	"math/big"

	"slices"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/defaults"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/numberdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/numberplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stoewer/go-strcase"
	"magalu.cloud/core"
	mgcSchemaPkg "magalu.cloud/core/schema"
	mgcSdk "magalu.cloud/sdk"
)

type mgcName string
type tfName string

type resAttrInfo struct {
	tfName             tfName
	mgcName            mgcName
	mgcSchema          *mgcSdk.Schema
	tfSchema           schema.Attribute
	currentCounterpart *resAttrInfo
	childAttributes    resAttrInfoMap
}

type resAttrInfoMap map[mgcName]*resAttrInfo

type resAttrInfoTree struct {
	input  resAttrInfoMap
	output resAttrInfoMap
}

type resAttrInfoGenMetadata struct {
	schema    *mgcSdk.Schema
	modifiers func(ctx context.Context, mgcSchema *mgcSdk.Schema, mgcName mgcName) attributeModifiers
}

type attributeModifiers struct {
	isRequired                 bool
	isOptional                 bool
	isComputed                 bool
	useStateForUnknown         bool
	requiresReplaceWhenChanged bool
	nameOverride               tfName
	getChildModifiers          func(ctx context.Context, mgcSchema *mgcSdk.Schema, mgcName mgcName) attributeModifiers
}

func addMgcSchemaAttributes(
	dst resAttrInfoMap,
	mgcSchema *mgcSdk.Schema,
	getModifiers func(ctx context.Context, mgcSchema *mgcSdk.Schema, mgcName mgcName) attributeModifiers,
	ctx context.Context,
) error {
	mgcSchemaTransformed, err := getXOfObjectSchemaTransformed(mgcSchema)
	if err != nil {
		return err
	}

	for propName, propSchemaRef := range mgcSchemaTransformed.Properties {
		tflog.SubsystemDebug(ctx, schemaGenSubsystem, fmt.Sprintf("adding attribute %q", propName))
		mgcName := mgcName(propName)
		propSchema := (*mgcSchemaPkg.Schema)(propSchemaRef.Value)

		if ca, ok := dst[mgcName]; ok {
			if err := mgcSchemaPkg.CompareJsonSchemas(ca.mgcSchema, propSchema); err != nil {
				// Ignore update value in favor of create value (This is probably a bug with the API)
				tflog.SubsystemError(ctx, schemaGenSubsystem, fmt.Sprintf("ignoring DIFFERENT attribute %q:\nOLD=%+v\nNEW=%+v\nERROR=%s\n", propName, ca.mgcSchema, propSchema, err.Error()))
				continue
			} else {
				tflog.SubsystemDebug(ctx, schemaGenSubsystem, fmt.Sprintf("ignoring already computed attribute %q ", propName))
				continue
			}
		}

		modifiers := getModifiers(ctx, mgcSchema, mgcName)
		if hasSchemaBeenPromoted(propSchema) {
			if modifiers.isRequired {
				modifiers.isRequired = false
			}
			if !modifiers.isComputed {
				modifiers.isOptional = true
			}
			tflog.SubsystemDebug(ctx, schemaGenSubsystem, fmt.Sprintf("computing %q as oneOf, %+v", propName, propSchema))
		}

		tfSchema, childAttributes, err := mgcSchemaToTFAttribute(propSchema, getModifiers(ctx, mgcSchema, mgcName), ctx)
		tflog.SubsystemDebug(ctx, schemaGenSubsystem, fmt.Sprintf("attribute %q generated tfSchema %#v", propName, tfSchema))
		if err != nil {
			tflog.SubsystemError(ctx, schemaGenSubsystem, fmt.Sprintf("attribute %q schema: %+v; error: %s", propName, propSchema, err))
			return fmt.Errorf("attribute %q, error=%s", propName, err)
		}

		name := mgcName.asTFName()
		if modifiers.nameOverride != "" {
			name = modifiers.nameOverride
		}

		attr := &resAttrInfo{
			tfName:          name,
			mgcName:         mgcName,
			mgcSchema:       propSchema,
			tfSchema:        tfSchema,
			childAttributes: childAttributes,
		}
		dst[mgcName] = attr
		tflog.SubsystemDebug(ctx, schemaGenSubsystem, fmt.Sprintf("attribute %q: %+v", propName, attr))
	}

	return nil
}

func getInputChildModifiers(ctx context.Context, mgcSchema *mgcSdk.Schema, mgcName mgcName) attributeModifiers {
	k := string(mgcName)
	isRequired := slices.Contains(mgcSchema.Required, k)
	return attributeModifiers{
		isRequired:                 isRequired,
		isOptional:                 !isRequired,
		isComputed:                 false, // This is being set to false because the parent may already be Computed, no further logic is needed here
		useStateForUnknown:         true,
		requiresReplaceWhenChanged: false,
		getChildModifiers:          getInputChildModifiers,
	}
}

func getResultModifiers(ctx context.Context, mgcSchema *mgcSdk.Schema, mgcName mgcName) attributeModifiers {
	return attributeModifiers{
		isRequired:                 false,
		isOptional:                 false,
		isComputed:                 true,
		useStateForUnknown:         false,
		requiresReplaceWhenChanged: false,
		getChildModifiers:          getResultModifiers,
	}
}

func generateResAttrInfoMap(ctx context.Context, resName tfName, metadatas []resAttrInfoGenMetadata, d *Diagnostics) resAttrInfoMap {
	ctx = tflog.SubsystemSetField(ctx, schemaGenSubsystem, resourceNameField, resName)
	tflog.SubsystemDebug(ctx, schemaGenSubsystem, "reading input attributes")

	attrInfoMap := resAttrInfoMap{}
	for _, metadata := range metadatas {
		err := addMgcSchemaAttributes(attrInfoMap, metadata.schema, metadata.modifiers, ctx)
		if err != nil {
			d.AddError("could not create TF input attributes", err.Error())
			return nil
		}
	}

	return attrInfoMap
}

func generateTFSchema(ctx context.Context, name tfName, description string, attrInfoTree resAttrInfoTree, d *Diagnostics) (tfSchema schema.Schema) {
	tflog.Debug(ctx, "generating schema")

	ctx = tflog.NewSubsystem(ctx, schemaGenSubsystem)
	ctx = tflog.SubsystemSetField(ctx, schemaGenSubsystem, resourceNameField, name)

	tfAttributes := generateTFAttributes(ctx, attrInfoTree, d)
	if d.HasError() {
		return
	}

	tfSchema = schema.Schema{Attributes: map[string]schema.Attribute{}}
	tfSchema.MarkdownDescription = description
	for tfName, tfAttr := range tfAttributes {
		tfSchema.Attributes[string(tfName)] = tfAttr
	}

	tfAttributeNames := []tfName{}
	for attrName := range tfAttributes {
		tfAttributeNames = append(tfAttributeNames, attrName)
	}

	tflog.Debug(ctx, "generated tf schema", map[string]any{"attributes": tfAttributeNames})

	return tfSchema
}

func generateTFAttributes(ctx context.Context, attrInfoTree resAttrInfoTree, d *Diagnostics) map[tfName]schema.Attribute {
	tflog.SubsystemInfo(ctx, schemaGenSubsystem, "reading input attributes")

	tfAttributes := map[tfName]schema.Attribute{}
	tflog.SubsystemInfo(ctx, schemaGenSubsystem, "generating attributes using input")
	for name, iattr := range attrInfoTree.input {
		// Split attributes that differ between input/output
		for _, oattr := range attrInfoTree.output {
			if iattr.tfName != oattr.tfName {
				continue
			}
			if err := mgcSchemaPkg.CompareJsonSchemas(oattr.mgcSchema, iattr.mgcSchema); err != nil {
				os, _ := oattr.mgcSchema.MarshalJSON()
				is, _ := iattr.mgcSchema.MarshalJSON()
				tflog.SubsystemDebug(ctx, schemaGenSubsystem, fmt.Sprintf("attribute %q differs between input and output. input: %s - output %s\nerror=%s", name, is, os, err.Error()))
				iattr.tfName = iattr.tfName.asDesired()
				oattr.tfName = oattr.tfName.asCurrent()

				iattr.currentCounterpart = oattr
			}
		}

		tfAttributes[iattr.tfName] = iattr.tfSchema
	}

	tflog.SubsystemInfo(ctx, schemaGenSubsystem, "generating attributes using output")
	for _, oattr := range attrInfoTree.output {
		// If they don't differ and it's already created skip
		if _, ok := tfAttributes[oattr.tfName]; ok {
			continue
		}

		tfAttributes[oattr.tfName] = oattr.tfSchema
	}

	return tfAttributes
}

func mgcSchemaToTFAttribute(mgcSchema *mgcSdk.Schema, m attributeModifiers, ctx context.Context) (schema.Attribute, resAttrInfoMap, error) {
	description := mgcSchema.Description

	switch mgcSchema.Type {
	case "string":
		return mgcStringSchemaToTFAttribute(ctx, description, mgcSchema, m)
	case "number":
		return mgcNumberSchemaToTFAttribute(ctx, description, mgcSchema, m)
	case "integer":
		return mgcIntSchemaToTFAttribute(ctx, description, mgcSchema, m)
	case "boolean":
		return mgcBoolSchemaToTFAttribute(ctx, description, mgcSchema, m)
	case "array":
		return mgcArraySchemaToTFAttribute(ctx, description, mgcSchema, m)
	case "object":
		return mgcObjectSchemaToTFAttribute(ctx, description, mgcSchema, m)
	default:
		return nil, nil, fmt.Errorf("type %q not supported", mgcSchema.Type)
	}
}

func mgcStringSchemaToTFAttribute(ctx context.Context, description string, mgcSchema *mgcSdk.Schema, m attributeModifiers) (schema.StringAttribute, resAttrInfoMap, error) {
	tflog.SubsystemDebug(ctx, schemaGenSubsystem, "generating attribute as string", map[string]any{"mgcSchema": mgcSchema})
	// I wanted to use an interface to define the modifiers regardless of the attr type
	// but couldn't find the interface, it seems everything is redefined for each type
	// https://github.com/hashicorp/terraform-plugin-framework/blob/main/internal/fwschema/fwxschema/attribute_plan_modification.go
	mod := []planmodifier.String{}
	if m.useStateForUnknown {
		mod = append(mod, stringplanmodifier.UseStateForUnknown())
	}
	if m.requiresReplaceWhenChanged {
		mod = append(mod, stringplanmodifier.RequiresReplace())
	}

	var d defaults.String
	if v, ok := mgcSchema.Default.(string); ok && m.isComputed {
		d = stringdefault.StaticString(v)
	}

	return schema.StringAttribute{
		Description:   description,
		Required:      m.isRequired,
		Optional:      m.isOptional,
		Computed:      m.isComputed,
		PlanModifiers: mod,
		Default:       d,
	}, nil, nil
}

func mgcNumberSchemaToTFAttribute(ctx context.Context, description string, mgcSchema *mgcSdk.Schema, m attributeModifiers) (schema.NumberAttribute, resAttrInfoMap, error) {
	tflog.SubsystemDebug(ctx, schemaGenSubsystem, "generating attribute as number", map[string]any{"mgcSchema": mgcSchema})
	mod := []planmodifier.Number{}
	if m.useStateForUnknown {
		mod = append(mod, numberplanmodifier.UseStateForUnknown())
	}
	if m.requiresReplaceWhenChanged {
		mod = append(mod, numberplanmodifier.RequiresReplace())
	}

	var d defaults.Number
	if v, ok := mgcSchema.Default.(float64); ok && m.isComputed {
		d = numberdefault.StaticBigFloat(big.NewFloat(v))
	}

	return schema.NumberAttribute{
		Description:   description,
		Required:      m.isRequired,
		Optional:      m.isOptional,
		Computed:      m.isComputed,
		PlanModifiers: mod,
		Default:       d,
	}, nil, nil
}

func mgcIntSchemaToTFAttribute(ctx context.Context, description string, mgcSchema *mgcSdk.Schema, m attributeModifiers) (schema.Int64Attribute, resAttrInfoMap, error) {
	tflog.SubsystemDebug(ctx, schemaGenSubsystem, "generating attribute as int", map[string]any{"mgcSchema": mgcSchema})
	mod := []planmodifier.Int64{}
	if m.useStateForUnknown {
		mod = append(mod, int64planmodifier.UseStateForUnknown())
	}
	if m.requiresReplaceWhenChanged {
		mod = append(mod, int64planmodifier.RequiresReplace())
	}

	var d defaults.Int64
	if v, ok := mgcSchema.Default.(int64); ok && m.isComputed {
		d = int64default.StaticInt64(v)
	}

	return schema.Int64Attribute{
		Description:   description,
		Required:      m.isRequired,
		Optional:      m.isOptional,
		Computed:      m.isComputed,
		PlanModifiers: mod,
		Default:       d,
	}, nil, nil
}

func mgcBoolSchemaToTFAttribute(ctx context.Context, description string, mgcSchema *mgcSdk.Schema, m attributeModifiers) (schema.BoolAttribute, resAttrInfoMap, error) {
	tflog.SubsystemDebug(ctx, schemaGenSubsystem, "generating attribute as bool", map[string]any{"mgcSchema": mgcSchema})
	mod := []planmodifier.Bool{}
	if m.useStateForUnknown {
		mod = append(mod, boolplanmodifier.UseStateForUnknown())
	}
	if m.requiresReplaceWhenChanged {
		mod = append(mod, boolplanmodifier.RequiresReplace())
	}

	var d defaults.Bool
	if v, ok := mgcSchema.Default.(bool); ok && m.isComputed {
		d = booldefault.StaticBool(v)
	}

	return schema.BoolAttribute{
		Description:   description,
		Required:      m.isRequired,
		Optional:      m.isOptional,
		Computed:      m.isComputed,
		PlanModifiers: mod,
		Default:       d,
	}, nil, nil
}

func mgcArraySchemaToTFAttribute(ctx context.Context, description string, mgcSchema *mgcSdk.Schema, m attributeModifiers) (schema.Attribute, resAttrInfoMap, error) {
	tflog.SubsystemDebug(ctx, schemaGenSubsystem, "generating attribute as array", map[string]any{"mgcSchema": mgcSchema})
	mgcItemSchema := (*core.Schema)(mgcSchema.Items.Value)
	elemAttr, elemAttrs, err := mgcSchemaToTFAttribute(mgcItemSchema, m.getChildModifiers(ctx, mgcItemSchema, "0"), ctx)
	if err != nil {
		return nil, nil, err
	}

	childAttrs := resAttrInfoMap{}
	childAttrs["0"] = &resAttrInfo{
		tfName:          "0",
		mgcName:         "0",
		mgcSchema:       mgcItemSchema,
		tfSchema:        elemAttr,
		childAttributes: elemAttrs,
	}

	mod := []planmodifier.List{}
	if m.requiresReplaceWhenChanged {
		mod = append(mod, listplanmodifier.RequiresReplace())
	}
	if m.useStateForUnknown {
		mod = append(mod, listplanmodifier.UseStateForUnknown())
	}

	var d defaults.List
	if v, ok := mgcSchema.Default.([]any); ok && m.isComputed {
		lst, err := tfAttrListValueFromMgcSchema(ctx, mgcSchema, *childAttrs["0"], v)
		if err != nil {
			return nil, nil, err
		}

		if l, ok := lst.(types.List); ok {
			d = listdefault.StaticValue(l)
		}
	}

	// TODO: How will we handle List of Lists? Does it need to be handled at all? Does the
	// 'else' branch already cover that correctly?
	if objAttr, ok := elemAttr.(schema.SingleNestedAttribute); ok {
		// This type assertion will/should NEVER fail, according to TF code
		nestedObj, ok := objAttr.GetNestedObject().(schema.NestedAttributeObject)
		if !ok {
			return nil, nil, fmt.Errorf("failed TF GetNestedObject")
		}
		return schema.ListNestedAttribute{
			NestedObject:  nestedObj,
			Description:   description,
			Required:      m.isRequired,
			Optional:      m.isOptional,
			Computed:      m.isComputed,
			PlanModifiers: mod,
			Default:       d,
		}, childAttrs, nil
	} else {
		return schema.ListAttribute{
			ElementType:   elemAttr.GetType(),
			Description:   description,
			Required:      m.isRequired,
			Optional:      m.isOptional,
			Computed:      m.isComputed,
			PlanModifiers: mod,
			Default:       d,
		}, childAttrs, nil
	}
}

func mgcObjectSchemaToTFAttribute(ctx context.Context, description string, mgcSchema *mgcSdk.Schema, m attributeModifiers) (schema.Attribute, resAttrInfoMap, error) {
	tflog.SubsystemDebug(ctx, schemaGenSubsystem, "generating attribute as object", map[string]any{"mgcSchema": mgcSchema})
	childAttrs := resAttrInfoMap{}
	err := addMgcSchemaAttributes(childAttrs, mgcSchema, m.getChildModifiers, ctx)
	if err != nil {
		return nil, nil, err
	}
	tfAttributes := map[string]schema.Attribute{}
	for _, attr := range childAttrs {
		tfAttributes[string(attr.tfName)] = attr.tfSchema
	}

	mod := []planmodifier.Object{}
	if m.requiresReplaceWhenChanged {
		mod = append(mod, objectplanmodifier.RequiresReplace())
	}
	if m.useStateForUnknown {
		mod = append(mod, objectplanmodifier.UseStateForUnknown())
	}

	var d defaults.Object
	if v, ok := mgcSchema.Default.(map[string]any); ok && m.isComputed {
		obj, err := tfAttrObjectValueFromMgcSchema(ctx, mgcSchema, childAttrs, v)
		if err != nil {
			return nil, nil, err
		}

		if o, ok := obj.(types.Object); ok {
			d = objectdefault.StaticValue(o)
		}
	}

	return schema.SingleNestedAttribute{
		Attributes:    tfAttributes,
		Description:   description,
		Required:      m.isRequired,
		Optional:      m.isOptional,
		Computed:      m.isComputed,
		PlanModifiers: mod,
		Default:       d,
	}, childAttrs, nil
}

func tfAttrListValueFromMgcSchema(ctx context.Context, s *mgcSdk.Schema, listAttr resAttrInfo, v []any) (attr.Value, error) {
	attrSchema := (*core.Schema)(s.Items.Value)
	attrType := listAttr.tfSchema.GetType()
	attrValues := []attr.Value{}
	for i := range v {
		v, ok, err := tfAttrValueFromMgcSchema(ctx, attrSchema, listAttr, v[i])
		if err != nil {
			return nil, err
		}

		if !ok {
			continue
		}

		attrValues = append(attrValues, v)
	}

	lst, diag := types.ListValue(attrType, attrValues)
	if diag.HasError() {
		return nil, fmt.Errorf("unable to create default list value")
	}
	return lst, nil
}

func tfAttrObjectValueFromMgcSchema(ctx context.Context, s *mgcSdk.Schema, mapAttr map[mgcName]*resAttrInfo, v map[string]any) (attr.Value, error) {
	attrTypes := map[string]attr.Type{}
	attrValues := map[string]attr.Value{}
	for k := range v {
		attrSchema := (*core.Schema)(s.Properties[k].Value)

		val, ok, err := tfAttrValueFromMgcSchema(ctx, attrSchema, *mapAttr[mgcName(k)], v[k])
		if err != nil {
			return nil, err
		}

		if !ok {
			continue
		}

		attrValues[k] = val
		attrTypes[k] = val.Type(ctx)
	}
	obj, diag := types.ObjectValue(attrTypes, attrValues)
	if diag.HasError() {
		return nil, fmt.Errorf("unable to create default object value")
	}
	return obj, nil
}

func tfAttrValueFromMgcSchema(ctx context.Context, s *mgcSdk.Schema, attrType resAttrInfo, v any) (attr.Value, bool, error) {
	if v == nil {
		return nil, false, nil
	}

	switch s.Type {
	case "string":
		if dStr, ok := v.(string); ok {
			return types.StringValue(dStr), true, nil
		}
		return nil, false, fmt.Errorf("unable to create attr.Value of type string")
	case "number":
		if dFloat, ok := v.(float64); ok {
			return types.NumberValue(big.NewFloat(dFloat)), true, nil
		}
		return nil, false, fmt.Errorf("unable to create attr.Value of type number")
	case "integer":
		if dInt, ok := v.(int64); ok {
			return types.Int64Value(dInt), true, nil
		}
		return nil, false, fmt.Errorf("unable to create attr.Value of type integer")
	case "boolean":
		if b, ok := v.(bool); ok {
			return types.BoolValue(b), true, nil
		}
		return nil, false, fmt.Errorf("unable to create attr.Value of type boolean")
	case "array":
		listVal, ok := v.([]any)
		if !ok {
			return nil, false, fmt.Errorf("unable to create attr.Value of type list")
		}

		attrValue, err := tfAttrListValueFromMgcSchema(ctx, s, attrType, listVal)
		if err != nil {
			return nil, false, err
		}
		return attrValue, true, nil
	case "object":
		mapVal, ok := v.(map[string]any)
		if !ok {
			return nil, false, fmt.Errorf("unable to create attr.Value of type object")
		}

		attrValue, err := tfAttrObjectValueFromMgcSchema(ctx, s, attrType.childAttributes, mapVal)
		if err != nil {
			return nil, false, err
		}
		return attrValue, true, nil
	default:
		return nil, false, fmt.Errorf("type %q not supported", s.Type)
	}
}

func (n mgcName) asTFName() tfName {
	return tfName(strcase.SnakeCase(string(n)))
}

func (n mgcName) tfNameOverride(r *MgcResource, s *mgcSdk.Schema) tfName {
	prefix := r.resMgcName.singular() + "_"

	target, found := strings.CutPrefix(string(n), string(prefix))
	if !found {
		return ""
	}

	if _, ok := r.read.ResultSchema().Properties[string(target)]; ok {
		return mgcName(target).asTFName()
	}

	return ""
}

func (n mgcName) singular() mgcName {
	if len(n) == 0 {
		return n
	}

	if n[len(n)-1] != 's' {
		return n
	}

	return n[:len(n)-1]
}

func (n tfName) asDesired() tfName {
	return n
}

func (n tfName) asCurrent() tfName {
	return "current_" + n
}
