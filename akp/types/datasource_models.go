package types

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

type InstanceDataSource struct {
	ID                     types.String                       `tfsdk:"id"`
	Name                   types.String                       `tfsdk:"name"`
	Workspace              types.String                       `tfsdk:"workspace"`
	ArgoCD                 *ArgoCDDataSource                  `tfsdk:"argocd"`
	ArgoCDConfigMap        types.Map                          `tfsdk:"argocd_cm"`
	ArgoCDRBACConfigMap    types.Map                          `tfsdk:"argocd_rbac_cm"`
	NotificationsConfigMap types.Map                          `tfsdk:"argocd_notifications_cm"`
	ImageUpdaterConfigMap  types.Map                          `tfsdk:"argocd_image_updater_config"`
	ImageUpdaterSSHConfig  types.Map                          `tfsdk:"argocd_image_updater_ssh_config"`
	ArgoCDKnownHosts       types.Map                          `tfsdk:"argocd_ssh_known_hosts_cm"`
	ArgoCDTLSCerts         types.Map                          `tfsdk:"argocd_tls_certs_cm"`
	ConfigManagementPlugin map[string]*ConfigManagementPlugin `tfsdk:"config_management_plugins"`
	ArgoCDResources        types.Map                          `tfsdk:"argocd_resources"`
}

type ArgoCDDataSource struct {
	Spec ArgoCDSpecDataSource `tfsdk:"spec"`
}

type ArgoCDSpecDataSource struct {
	Description  types.String           `tfsdk:"description"`
	Version      types.String           `tfsdk:"version"`
	InstanceSpec InstanceSpecDataSource `tfsdk:"instance_spec"`
}

type InstanceSpecDataSource struct {
	Subdomain                       types.String                   `tfsdk:"subdomain"`
	DeclarativeManagementEnabled    types.Bool                     `tfsdk:"declarative_management_enabled"`
	Extensions                      []*ArgoCDExtensionInstallEntry `tfsdk:"extensions"`
	ClusterCustomizationDefaults    types.Object                   `tfsdk:"cluster_customization_defaults"`
	ImageUpdaterEnabled             types.Bool                     `tfsdk:"image_updater_enabled"`
	BackendIpAllowListEnabled       types.Bool                     `tfsdk:"backend_ip_allow_list_enabled"`
	RepoServerDelegate              *RepoServerDelegate            `tfsdk:"repo_server_delegate"`
	AuditExtensionEnabled           types.Bool                     `tfsdk:"audit_extension_enabled"`
	SyncHistoryExtensionEnabled     types.Bool                     `tfsdk:"sync_history_extension_enabled"`
	CrossplaneExtension             *CrossplaneExtension           `tfsdk:"crossplane_extension"`
	ImageUpdaterDelegate            *ImageUpdaterDelegate          `tfsdk:"image_updater_delegate"`
	AppSetDelegate                  *AppSetDelegate                `tfsdk:"app_set_delegate"`
	AssistantExtensionEnabled       types.Bool                     `tfsdk:"assistant_extension_enabled"`
	AppsetPolicy                    types.Object                   `tfsdk:"appset_policy"`
	HostAliases                     []*HostAliases                 `tfsdk:"host_aliases"`
	AgentPermissionsRules           []*AgentPermissionsRule        `tfsdk:"agent_permissions_rules"`
	Fqdn                            types.String                   `tfsdk:"fqdn"`
	MultiClusterK8SDashboardEnabled types.Bool                     `tfsdk:"multi_cluster_k8s_dashboard_enabled"`
	AkuityIntelligenceExtension     *AkuityIntelligenceExtension   `tfsdk:"akuity_intelligence_extension"`
	KubeVisionConfig                *KubeVisionConfig              `tfsdk:"kube_vision_config"`
	AppInAnyNamespaceConfig         *AppInAnyNamespaceConfig       `tfsdk:"app_in_any_namespace_config"`
	AppsetPlugins                   []*AppsetPlugins               `tfsdk:"appset_plugins"`
	ApplicationSetExtension         *ApplicationSetExtension       `tfsdk:"application_set_extension"`
	MetricsIngressUsername          types.String                   `tfsdk:"metrics_ingress_username"`
	PrivilegedNotificationCluster   types.String                   `tfsdk:"privileged_notification_cluster"`
	ClusterAddonsExtension          *ClusterAddonsExtension        `tfsdk:"cluster_addons_extension"`
	ManifestGeneration              *ManifestGeneration            `tfsdk:"manifest_generation"`
}

func NewInstanceDataSourceModel(instance *Instance) InstanceDataSource {
	model := InstanceDataSource{
		ID:                     instance.ID,
		Name:                   instance.Name,
		Workspace:              instance.Workspace,
		ArgoCDConfigMap:        normalizeStringMap(instance.ArgoCDConfigMap),
		ArgoCDRBACConfigMap:    normalizeStringMap(instance.ArgoCDRBACConfigMap),
		NotificationsConfigMap: normalizeStringMap(instance.NotificationsConfigMap),
		ImageUpdaterConfigMap:  normalizeStringMap(instance.ImageUpdaterConfigMap),
		ImageUpdaterSSHConfig:  normalizeStringMap(instance.ImageUpdaterSSHConfigMap),
		ArgoCDKnownHosts:       normalizeStringMap(instance.ArgoCDKnownHostsConfigMap),
		ArgoCDTLSCerts:         normalizeStringMap(instance.ArgoCDTLSCertsConfigMap),
		ConfigManagementPlugin: instance.ConfigManagementPlugins,
		ArgoCDResources:        normalizeStringMap(instance.ArgoCDResources),
	}
	if instance.ArgoCD != nil {
		model.ArgoCD = &ArgoCDDataSource{
			Spec: ArgoCDSpecDataSource{
				Description: instance.ArgoCD.Spec.Description,
				Version:     instance.ArgoCD.Spec.Version,
				InstanceSpec: InstanceSpecDataSource{
					Subdomain:                       instance.ArgoCD.Spec.InstanceSpec.Subdomain,
					DeclarativeManagementEnabled:    instance.ArgoCD.Spec.InstanceSpec.DeclarativeManagementEnabled,
					Extensions:                      instance.ArgoCD.Spec.InstanceSpec.Extensions,
					ClusterCustomizationDefaults:    instance.ArgoCD.Spec.InstanceSpec.ClusterCustomizationDefaults,
					ImageUpdaterEnabled:             instance.ArgoCD.Spec.InstanceSpec.ImageUpdaterEnabled,
					BackendIpAllowListEnabled:       instance.ArgoCD.Spec.InstanceSpec.BackendIpAllowListEnabled,
					RepoServerDelegate:              instance.ArgoCD.Spec.InstanceSpec.RepoServerDelegate,
					AuditExtensionEnabled:           instance.ArgoCD.Spec.InstanceSpec.AuditExtensionEnabled,
					SyncHistoryExtensionEnabled:     instance.ArgoCD.Spec.InstanceSpec.SyncHistoryExtensionEnabled,
					CrossplaneExtension:             instance.ArgoCD.Spec.InstanceSpec.CrossplaneExtension,
					ImageUpdaterDelegate:            instance.ArgoCD.Spec.InstanceSpec.ImageUpdaterDelegate,
					AppSetDelegate:                  instance.ArgoCD.Spec.InstanceSpec.AppSetDelegate,
					AssistantExtensionEnabled:       instance.ArgoCD.Spec.InstanceSpec.AssistantExtensionEnabled,
					AppsetPolicy:                    instance.ArgoCD.Spec.InstanceSpec.AppsetPolicy,
					HostAliases:                     instance.ArgoCD.Spec.InstanceSpec.HostAliases,
					AgentPermissionsRules:           instance.ArgoCD.Spec.InstanceSpec.AgentPermissionsRules,
					Fqdn:                            instance.ArgoCD.Spec.InstanceSpec.Fqdn,
					MultiClusterK8SDashboardEnabled: instance.ArgoCD.Spec.InstanceSpec.MultiClusterK8SDashboardEnabled,
					AkuityIntelligenceExtension:     instance.ArgoCD.Spec.InstanceSpec.AkuityIntelligenceExtension,
					KubeVisionConfig:                instance.ArgoCD.Spec.InstanceSpec.KubeVisionConfig,
					AppInAnyNamespaceConfig:         instance.ArgoCD.Spec.InstanceSpec.AppInAnyNamespaceConfig,
					AppsetPlugins:                   instance.ArgoCD.Spec.InstanceSpec.AppsetPlugins,
					ApplicationSetExtension:         instance.ArgoCD.Spec.InstanceSpec.ApplicationSetExtension,
					MetricsIngressUsername:          instance.ArgoCD.Spec.InstanceSpec.MetricsIngressUsername,
					PrivilegedNotificationCluster:   instance.ArgoCD.Spec.InstanceSpec.PrivilegedNotificationCluster,
					ClusterAddonsExtension:          instance.ArgoCD.Spec.InstanceSpec.ClusterAddonsExtension,
					ManifestGeneration:              instance.ArgoCD.Spec.InstanceSpec.ManifestGeneration,
				},
			},
		}
	}
	return model
}

type KargoInstanceDataSource struct {
	ID             types.String     `tfsdk:"id"`
	Name           types.String     `tfsdk:"name"`
	Kargo          *KargoDataSource `tfsdk:"kargo"`
	KargoConfigMap types.Map        `tfsdk:"kargo_cm"`
	Workspace      types.String     `tfsdk:"workspace"`
	KargoResources types.Map        `tfsdk:"kargo_resources"`
}

type KargoDataSource struct {
	Spec KargoSpecDataSource `tfsdk:"spec"`
}

type KargoSpecDataSource struct {
	Description       types.String               `tfsdk:"description"`
	Version           types.String               `tfsdk:"version"`
	KargoInstanceSpec KargoInstanceSpec          `tfsdk:"kargo_instance_spec"`
	Fqdn              types.String               `tfsdk:"fqdn"`
	Subdomain         types.String               `tfsdk:"subdomain"`
	OidcConfig        *KargoOidcConfigDataSource `tfsdk:"oidc_config"`
}

type KargoOidcConfigDataSource struct {
	Enabled               types.Bool                  `tfsdk:"enabled"`
	DexEnabled            types.Bool                  `tfsdk:"dex_enabled"`
	DexConfig             types.String                `tfsdk:"dex_config"`
	IssuerURL             types.String                `tfsdk:"issuer_url"`
	ClientID              types.String                `tfsdk:"client_id"`
	CliClientID           types.String                `tfsdk:"cli_client_id"`
	AdminAccount          *KargoPredefinedAccountData `tfsdk:"admin_account"`
	ViewerAccount         *KargoPredefinedAccountData `tfsdk:"viewer_account"`
	AdditionalScopes      []types.String              `tfsdk:"additional_scopes"`
	UserAccount           *KargoPredefinedAccountData `tfsdk:"user_account"`
	ProjectCreatorAccount *KargoPredefinedAccountData `tfsdk:"project_creator_account"`
}

func NewKargoInstanceDataSourceModel(instance *KargoInstance) KargoInstanceDataSource {
	model := KargoInstanceDataSource{
		ID:             instance.ID,
		Name:           instance.Name,
		KargoConfigMap: normalizeStringMap(instance.KargoConfigMap),
		Workspace:      instance.Workspace,
		KargoResources: normalizeStringMap(instance.KargoResources),
	}
	if instance.Kargo != nil {
		model.Kargo = &KargoDataSource{
			Spec: KargoSpecDataSource{
				Description:       instance.Kargo.Spec.Description,
				Version:           instance.Kargo.Spec.Version,
				KargoInstanceSpec: instance.Kargo.Spec.KargoInstanceSpec,
				Fqdn:              instance.Kargo.Spec.Fqdn,
				Subdomain:         instance.Kargo.Spec.Subdomain,
			},
		}
		if instance.Kargo.Spec.OidcConfig != nil {
			model.Kargo.Spec.OidcConfig = &KargoOidcConfigDataSource{
				Enabled:               instance.Kargo.Spec.OidcConfig.Enabled,
				DexEnabled:            instance.Kargo.Spec.OidcConfig.DexEnabled,
				DexConfig:             instance.Kargo.Spec.OidcConfig.DexConfig,
				IssuerURL:             instance.Kargo.Spec.OidcConfig.IssuerURL,
				ClientID:              instance.Kargo.Spec.OidcConfig.ClientID,
				CliClientID:           instance.Kargo.Spec.OidcConfig.CliClientID,
				AdminAccount:          instance.Kargo.Spec.OidcConfig.AdminAccount,
				ViewerAccount:         instance.Kargo.Spec.OidcConfig.ViewerAccount,
				AdditionalScopes:      instance.Kargo.Spec.OidcConfig.AdditionalScopes,
				UserAccount:           instance.Kargo.Spec.OidcConfig.UserAccount,
				ProjectCreatorAccount: instance.Kargo.Spec.OidcConfig.ProjectCreatorAccount,
			}
		}
	}
	return model
}

func normalizeStringMap(value types.Map) types.Map {
	if value.ElementType(context.Background()) == nil {
		return types.MapNull(types.StringType)
	}
	return value
}
