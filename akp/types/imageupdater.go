package types

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"gopkg.in/yaml.v3"
)

type AkpImageUpdater struct {
	Secrets     types.Map    `tfsdk:"secrets"`
	SshConfig   types.String `tfsdk:"ssh_config"`
	GitUser     types.String `tfsdk:"git_user"`
	GitEmail    types.String `tfsdk:"git_email"`
	GitTemplate types.String `tfsdk:"git_template"`
	LogLevel    types.String `tfsdk:"log_level"`
	Registries  types.Map    `tfsdk:"registries"`
}

var (
	imageUpdaterAttrTypes = map[string]attr.Type{
		"secrets": types.MapType{
			ElemType: types.ObjectType{AttrTypes: secretAttrTypes},
		},
		"ssh_config":   types.StringType,
		"git_user":     types.StringType,
		"git_email":    types.StringType,
		"git_template": types.StringType,
		"log_level":    types.StringType,
		"registries": types.MapType{
			ElemType: types.ObjectType{
				AttrTypes: imageUpdaterRegistryAttrTypes,
			},
		},
	}
)

func MergeImageUpdater(state *AkpImageUpdater, plan *AkpImageUpdater) (*AkpImageUpdater, diag.Diagnostics) {
	diags := diag.Diagnostics{}
	res := &AkpImageUpdater{}

	if plan.Secrets.IsUnknown() {
		res.Secrets = state.Secrets
	} else {
		secrets, d := MergeSecrets(&state.Secrets, &plan.Secrets)
		diags.Append(d...)
		res.Secrets = *secrets
	}

	if plan.SshConfig.IsUnknown() {
		res.SshConfig = state.SshConfig
	} else if plan.SshConfig.IsNull() {
		res.SshConfig = types.StringNull()
	} else {
		res.SshConfig = plan.SshConfig
	}

	if plan.GitUser.IsUnknown() {
		res.GitUser = state.GitUser
	} else if plan.GitUser.IsNull() {
		res.GitUser = types.StringNull()
	} else {
		res.GitUser = plan.GitUser
	}

	if plan.GitEmail.IsUnknown() {
		res.GitEmail = state.GitEmail
	} else if plan.GitEmail.IsNull() {
		res.GitEmail = types.StringNull()
	} else {
		res.GitEmail = plan.GitEmail
	}

	if plan.GitTemplate.IsUnknown() {
		res.GitTemplate = state.GitTemplate
	} else if plan.GitTemplate.IsNull() {
		res.GitTemplate = types.StringNull()
	} else {
		res.GitTemplate = plan.GitTemplate
	}

	if plan.LogLevel.IsUnknown() {
		res.LogLevel = state.LogLevel
	} else if plan.LogLevel.IsNull() {
		res.LogLevel = types.StringNull()
	} else {
		res.LogLevel = plan.LogLevel
	}

	if plan.Registries.IsUnknown() {
		res.Registries = state.Registries
	} else {
		registries, d := MergeRegistries(&state.Registries, &plan.Registries)
		diags.Append(d...)
		res.Registries = *registries
	}

	return res, diags
}

func (x *AkpImageUpdater) UpdateImageUpdater(iuSecrets map[string]string, iuConfig map[string]string, iuSshConfig map[string]string) diag.Diagnostics {
	var d diag.Diagnostics
	diags := diag.Diagnostics{}
	if len(iuSecrets) == 0 {
		x.Secrets = types.MapNull(types.ObjectType{AttrTypes: secretAttrTypes}) // not computed => can be null
	} else {
		x.Secrets, d = MapValueFromMap(iuSecrets)
		diags.Append(d...)
	}

	sshConfig, ok := iuSshConfig["config"]
	if ok {
		x.SshConfig = types.StringValue(sshConfig)
	} else {
		x.SshConfig = types.StringNull()
	}

	gitUser, ok := iuConfig["git.user"]
	if ok {
		x.GitUser = types.StringValue(gitUser)
	} else {
		x.GitUser = types.StringNull()
	}

	gitEmail, ok := iuConfig["git.email"]
	if ok {
		x.GitEmail = types.StringValue(gitEmail)
	} else {
		x.GitEmail = types.StringNull()
	}

	gitTemplate, ok := iuConfig["git.commit-message-template"]
	if ok {
		x.GitTemplate = types.StringValue(gitTemplate)
	} else {
		x.GitTemplate = types.StringNull()
	}

	logLevel, ok := iuConfig["log.level"]
	if ok {
		x.LogLevel = types.StringValue(logLevel)
	} else {
		x.LogLevel = types.StringNull()
	}

	registriesTxt, ok := iuConfig["registries.conf"]
	if ok {
		var registriesYaml ImageUpdaterRegistriesYaml
		err := yaml.Unmarshal([]byte(registriesTxt), &registriesYaml)
		if err != nil {
			diags.AddError("Yaml Error", err.Error())
			return diags
		}
		if len(registriesYaml.Registries) != 0 {
			registries := make(map[string]*AkpImageUpdaterRegistry)
			RegistriesYamlAsTf(registriesYaml, &registries)
			x.Registries, d = types.MapValueFrom(context.Background(), types.ObjectType{AttrTypes: imageUpdaterRegistryAttrTypes}, registries)
		} else {
			x.Registries = types.MapNull(types.ObjectType{AttrTypes: imageUpdaterRegistryAttrTypes})
		}
	} else {
		x.Registries = types.MapNull(types.ObjectType{AttrTypes: imageUpdaterRegistryAttrTypes})
	}
	diags.Append(d...)
	return diags
}

func (x *AkpImageUpdater) PopulateSecrets(source *AkpImageUpdater) {
	secrets, _ := MapFromMapValue(x.Secrets)
	sourceSecrets, _ := MapFromMapValue(source.Secrets)
	for name := range secrets {
		secrets[name] = sourceSecrets[name]
	}
	x.Secrets, _ = MapValueFromMap(secrets)

}

func (x *AkpImageUpdater) GetSensitiveStrings() []string {
	var res []string
	secrets, _ := MapFromMapValue(x.Secrets)
	for _, value := range secrets {
		res = append(res, value)
	}
	return res
}

func (x *AkpImageUpdater) ConfigAsMap() map[string]string {
	res := make(map[string]string)

	if !x.GitUser.IsNull() {
		res["git.user"] = x.GitUser.ValueString()
	}
	if !x.GitEmail.IsNull() {
		res["git.email"] = x.GitEmail.ValueString()
	}
	if !x.GitTemplate.IsNull() {
		res["git.commit-message-template"] = x.GitTemplate.ValueString()
	}
	if !x.LogLevel.IsNull() {
		res["log.level"] = x.LogLevel.ValueString()
	}
	if !x.Registries.IsNull() {
		var registriesTf map[string]*AkpImageUpdaterRegistry
		registriesYaml := ImageUpdaterRegistriesYaml{}
		x.Registries.ElementsAs(context.Background(), &registriesTf, true)
		RegistriesTfAsYaml(registriesTf, &registriesYaml)
		registriesTxt, _ := yaml.Marshal(registriesYaml)
		res["registries.conf"] = string(registriesTxt)
	}
	return res
}

func (x *AkpImageUpdater) SshConfigAsMap() map[string]string {
	res := make(map[string]string)
	if !x.SshConfig.IsNull() {
		res["config"] = x.SshConfig.ValueString()
	}
	return res
}
