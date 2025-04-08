// This is an auto-generated file. DO NOT EDIT
/*
Copyright 2025 Akuity, Inc.
*/

package types

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type KargoAgent struct {
	ID                            types.String    `tfsdk:"id"`
	InstanceID                    types.String    `tfsdk:"instance_id"`
	Name                          types.String    `tfsdk:"name"`
	Labels                        types.Map       `tfsdk:"labels"`
	Annotations                   types.Map       `tfsdk:"annotations"`
	Spec                          *KargoAgentSpec `tfsdk:"spec"`
	Kubeconfig                    *Kubeconfig     `tfsdk:"kube_config"`
	RemoveAgentResourcesOnDestroy types.Bool      `tfsdk:"remove_agent_resources_on_destroy"`
}

type KargoAgents struct {
	ID         types.String `tfsdk:"id"`
	InstanceID types.String `tfsdk:"instance_id"`
	Agents     []KargoAgent `tfsdk:"agents"`
}

type KargoAgentSpec struct {
	Description types.String   `tfsdk:"description"`
	Data        KargoAgentData `tfsdk:"data"`
}

type KargoAgentData struct {
	Size                types.String `tfsdk:"size"`
	AutoUpgradeDisabled types.Bool   `tfsdk:"auto_upgrade_disabled"`
	TargetVersion       types.String `tfsdk:"target_version"`
	Kustomization       types.String `tfsdk:"kustomization"`
	RemoteArgocd        types.String `tfsdk:"remote_argocd"`
	AkuityManaged       types.Bool   `tfsdk:"akuity_managed"`
	ArgocdNamespace     types.String `tfsdk:"argocd_namespace"`
}
