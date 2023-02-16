package types

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type AkpImageUpdaterRegistry struct {
	Prefix      types.String `tfsdk:"prefix"`
	ApiUrl      types.String `tfsdk:"api_url"`
	DefaultNs   types.String `tfsdk:"defaultns"`
	Credentials types.String `tfsdk:"credentials"`
	CredsExpire types.String `tfsdk:"credsexpire"`
	Limit       types.String `tfsdk:"limit"`
	Default     types.Bool   `tfsdk:"default"`
	Insecure    types.Bool   `tfsdk:"insecure"`
}

type ImageUpdaterRegistryYaml struct {
	Prefix      string  `yaml:"prefix"`
	Name        string  `yaml:"name"`
	ApiUrl      *string `yaml:"api_url"`
	DefaultNs   *string `yaml:"defaultns"`
	Credentials *string `yaml:"credentials"`
	CredsExpire *string `yaml:"credsexpire"`
	Limit       *string `yaml:"limit"`
	Default     *bool   `yaml:"default"`
	Insecure    *bool   `yaml:"insecure"`
}

type ImageUpdaterRegistriesYaml struct {
	Registries []ImageUpdaterRegistryYaml `yaml:"registries"`
}

var (
	imageUpdaterRegistryAttrTypes = map[string]attr.Type{
		"prefix":      types.StringType,
		"api_url":     types.StringType,
		"defaultns":   types.StringType,
		"credentials": types.StringType,
		"credsexpire": types.StringType,
		"limit":       types.StringType,
		"default":     types.BoolType,
		"insecure":    types.BoolType,
	}
)

func MergeRegistries(state *types.Map, plan *types.Map) (*types.Map, diag.Diagnostics) {
	var stateRegistries, planRegistries map[string]AkpImageUpdaterRegistry
	diags := diag.Diagnostics{}
	if !state.IsNull() {
		diags.Append(state.ElementsAs(context.Background(), &stateRegistries, true)...)
	} else {
		stateRegistries = make(map[string]AkpImageUpdaterRegistry)
	}
	if !plan.IsNull() && !plan.IsUnknown() {
		diags.Append(plan.ElementsAs(context.Background(), &planRegistries, true)...)
	} else {
		planRegistries = make(map[string]AkpImageUpdaterRegistry)
	}
	res := make(map[string]AkpImageUpdaterRegistry)
	for name := range stateRegistries {
		if val, ok := planRegistries[name]; ok {
			res[name] = val // update registry from plan
		}
	}
	for name := range planRegistries {
		if _, ok := stateRegistries[name]; !ok {
			res[name] = planRegistries[name]
		}
	}
	resMap, d := types.MapValueFrom(context.Background(), types.ObjectType{AttrTypes: imageUpdaterRegistryAttrTypes}, res)
	return &resMap, d
}

func RegistriesYamlAsTf(registriesYaml ImageUpdaterRegistriesYaml, registries *map[string]*AkpImageUpdaterRegistry) error {
	target := *registries
	for _, registryYaml := range registriesYaml.Registries {
		target[registryYaml.Name] = &AkpImageUpdaterRegistry{
			Prefix: types.StringValue(registryYaml.Prefix),
		}
		if registryYaml.ApiUrl == nil {
			target[registryYaml.Name].ApiUrl = types.StringNull()
		} else {
			target[registryYaml.Name].ApiUrl = types.StringValue(*registryYaml.ApiUrl)
		}

		if registryYaml.DefaultNs == nil {
			target[registryYaml.Name].DefaultNs = types.StringNull()
		} else {
			target[registryYaml.Name].DefaultNs = types.StringValue(*registryYaml.DefaultNs)
		}

		if registryYaml.Credentials == nil {
			target[registryYaml.Name].Credentials = types.StringNull()
		} else {
			target[registryYaml.Name].Credentials = types.StringValue(*registryYaml.Credentials)
		}

		if registryYaml.CredsExpire == nil {
			target[registryYaml.Name].CredsExpire = types.StringNull()
		} else {
			target[registryYaml.Name].CredsExpire = types.StringValue(*registryYaml.CredsExpire)
		}

		if registryYaml.Limit == nil {
			target[registryYaml.Name].Limit = types.StringNull()
		} else {
			target[registryYaml.Name].Limit = types.StringValue(*registryYaml.Limit)
		}

		if registryYaml.Default == nil {
			target[registryYaml.Name].Default = types.BoolNull()
		} else {
			target[registryYaml.Name].Default = types.BoolValue(*registryYaml.Default)
		}

		if registryYaml.Insecure == nil {
			target[registryYaml.Name].Insecure = types.BoolNull()
		} else {
			target[registryYaml.Name].Insecure = types.BoolValue(*registryYaml.Insecure)
		}
	}
	return nil
}

func RegistriesTfAsYaml(registries map[string]*AkpImageUpdaterRegistry, target *ImageUpdaterRegistriesYaml) error {
	for name, registry := range registries {
		registryYaml := ImageUpdaterRegistryYaml{
			Name:   name,
			Prefix: registry.Prefix.ValueString(),
		}
		if !registry.ApiUrl.IsNull() {
			s := registry.ApiUrl.ValueString()
			registryYaml.ApiUrl = &s
		}
		if !registry.DefaultNs.IsNull() {
			s := registry.DefaultNs.ValueString()
			registryYaml.DefaultNs = &s
		}
		if !registry.Credentials.IsNull() {
			s := registry.Credentials.ValueString()
			registryYaml.Credentials = &s
		}
		if !registry.CredsExpire.IsNull() {
			s := registry.CredsExpire.ValueString()
			registryYaml.CredsExpire = &s
		}
		if !registry.Limit.IsNull() {
			s := registry.Limit.ValueString()
			registryYaml.Limit = &s
		}
		if !registry.Default.IsNull() {
			s := registry.Default.ValueBool()
			registryYaml.Default = &s
		}
		if !registry.Insecure.IsNull() {
			s := registry.Insecure.ValueBool()
			registryYaml.Insecure = &s
		}
		target.Registries = append(target.Registries, registryYaml)
	}
	return nil
}
