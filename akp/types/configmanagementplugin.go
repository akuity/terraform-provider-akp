// This is an auto-generated file. DO NOT EDIT
/*
Copyright 2023 Akuity, Inc.
*/

package types

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type ConfigManagementPlugin struct {
	Name       types.String `tfsdk:"name"`
	InstanceID types.String `tfsdk:"instance_id"`
	Enabled    types.Bool   `tfsdk:"enabled"`
	Image      types.String `tfsdk:"image"`
	Spec       *PluginSpec  `tfsdk:"spec"`
}

type ConfigManagementPlugins struct {
	InstanceID types.String             `tfsdk:"instance_id"`
	Plugins    []ConfigManagementPlugin `tfsdk:"plugins"`
}

type PluginSpec struct {
	Version          types.String `tfsdk:"version"`
	Init             *Command     `tfsdk:"init"`
	Generate         *Command     `tfsdk:"generate"`
	Discover         *Discover    `tfsdk:"discover"`
	Parameters       *Parameters  `tfsdk:"parameters"`
	PreserveFileMode types.Bool   `tfsdk:"preserve_file_mode"`
}

type Command struct {
	Command []types.String `tfsdk:"command"`
	Args    []types.String `tfsdk:"args"`
}

type Discover struct {
	Find     *Find        `tfsdk:"find"`
	FileName types.String `tfsdk:"file_name"`
}

type Find struct {
	Command []types.String `tfsdk:"command"`
	Args    []types.String `tfsdk:"args"`
	Glob    types.String   `tfsdk:"glob"`
}

type Parameters struct {
	Static  []*ParameterAnnouncement `tfsdk:"static"`
	Dynamic *Dynamic                 `tfsdk:"dynamic"`
}

type Dynamic struct {
	Command []types.String `tfsdk:"command"`
	Args    []types.String `tfsdk:"args"`
}

type ParameterAnnouncement struct {
	Name           types.String   `tfsdk:"name"`
	Title          types.String   `tfsdk:"title"`
	Tooltip        types.String   `tfsdk:"tooltip"`
	Required       types.Bool     `tfsdk:"required"`
	ItemType       types.String   `tfsdk:"item_type"`
	CollectionType types.String   `tfsdk:"collection_type"`
	String_        types.String   `tfsdk:"string"`
	Array          []types.String `tfsdk:"array"`
	Map            types.Map      `tfsdk:"map"`
}
