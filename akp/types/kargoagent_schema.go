// This is an auto-generated file. DO NOT EDIT
/*
Copyright 2025 Akuity, Inc.
*/

package types

import (
	dataschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	resourceschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	KargoAgentResourceSchema resourceschema.Schema = resourceschema.Schema{
		Attributes: map[string]resourceschema.Attribute{
			"id": resourceschema.StringAttribute{Computed: true},
			"instance_id": resourceschema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			}, "workspace": resourceschema.StringAttribute{
				Optional: true, Computed: true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			}, "name": resourceschema.StringAttribute{Required: true, PlanModifiers: []planmodifier.String{
				stringplanmodifier.RequiresReplace(),
			}}, "namespace": resourceschema.StringAttribute{
				Optional: true, Computed: true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			}, "labels": resourceschema.MapAttribute{
				Optional: true, Computed: true,
				ElementType: types.StringType, PlanModifiers: []planmodifier.Map{
					mapplanmodifier.UseStateForUnknown(),
				},
			}, "annotations": resourceschema.MapAttribute{
				Optional: true, Computed: true,
				ElementType: types.StringType, PlanModifiers: []planmodifier.Map{
					mapplanmodifier.UseStateForUnknown(),
				},
			}, "spec": resourceschema.SingleNestedAttribute{Attributes: map[string]resourceschema.Attribute{
				"description": resourceschema.StringAttribute{
					Optional: true, Computed: true,
					PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				}, "data": resourceschema.SingleNestedAttribute{Attributes: map[string]resourceschema.Attribute{
					"size": resourceschema.StringAttribute{Required: true}, "auto_upgrade_disabled": resourceschema.BoolAttribute{
						Optional: true, Computed: true, PlanModifiers: []planmodifier.Bool{
							boolplanmodifier.UseStateForUnknown(),
						},
					}, "target_version": resourceschema.StringAttribute{
						Optional: true, Computed: true,
						PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
					}, "kustomization": resourceschema.StringAttribute{
						Optional: true, Computed: true,
						PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
					}, "remote_argocd": resourceschema.StringAttribute{
						Optional: true, Computed: true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
							stringplanmodifier.RequiresReplaceIfConfigured(),
						},
					}, "akuity_managed": resourceschema.BoolAttribute{
						Optional: true, Computed: true,
						PlanModifiers: []planmodifier.Bool{
							boolplanmodifier.UseStateForUnknown(),
							boolplanmodifier.RequiresReplaceIfConfigured(),
						},
					}, "argocd_namespace": resourceschema.StringAttribute{
						Optional: true, Computed: true,
						PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
					}, "self_managed_argocd_url": resourceschema.StringAttribute{
						Optional: true, Computed: true,
						PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
					},
				}, Required: true},
			}, Required: true}, "kube_config": resourceschema.SingleNestedAttribute{
				Attributes: map[string]resourceschema.Attribute{
					"host": resourceschema.StringAttribute{
						Optional: true,
					}, "username": resourceschema.StringAttribute{Optional: true},
					"password": resourceschema.StringAttribute{Optional: true, Sensitive: true},
					"insecure": resourceschema.BoolAttribute{Optional: true}, "client_certificate": resourceschema.StringAttribute{
						Optional: true,
					}, "client_key": resourceschema.StringAttribute{Optional: true, Sensitive: true},
					"cluster_ca_certificate":   resourceschema.StringAttribute{Optional: true},
					"config_path":              resourceschema.StringAttribute{Optional: true},
					"config_paths":             resourceschema.ListAttribute{Optional: true, ElementType: types.StringType},
					"config_context":           resourceschema.StringAttribute{Optional: true},
					"config_context_auth_info": resourceschema.StringAttribute{Optional: true},
					"config_context_cluster":   resourceschema.StringAttribute{Optional: true},
					"token":                    resourceschema.StringAttribute{Optional: true, Sensitive: true},
					"proxy_url":                resourceschema.StringAttribute{Optional: true},
					"exec": resourceschema.SingleNestedAttribute{Attributes: map[string]resourceschema.Attribute{
						"api_version": resourceschema.StringAttribute{
							Optional: true, Computed: true,
							PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
						}, "command": resourceschema.StringAttribute{
							Optional: true, Computed: true,
							PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
						}, "args": resourceschema.ListAttribute{Optional: true, ElementType: types.StringType},
						"env": resourceschema.MapAttribute{Optional: true, ElementType: types.StringType},
					}, Optional: true, PlanModifiers: []planmodifier.Object{objectplanmodifier.UseStateForUnknown()}},
				}, Optional: true, PlanModifiers: []planmodifier.Object{objectplanmodifier.UseStateForUnknown()},
			}, "remove_agent_resources_on_destroy": resourceschema.BoolAttribute{Optional: true, Computed: true},
		},
	}
	KargoAgentDataSourceSchema dataschema.Schema = dataschema.Schema{
		Attributes: map[string]dataschema.Attribute{
			"id":          dataschema.StringAttribute{Computed: true},
			"instance_id": dataschema.StringAttribute{Required: true}, "workspace": dataschema.StringAttribute{
				Computed: true,
			}, "name": dataschema.StringAttribute{Required: true}, "namespace": dataschema.StringAttribute{
				Computed: true,
			}, "labels": dataschema.MapAttribute{Computed: true, ElementType: types.StringType},
			"annotations": dataschema.MapAttribute{Computed: true, ElementType: types.StringType},
			"spec": dataschema.SingleNestedAttribute{Attributes: map[string]dataschema.Attribute{
				"description": dataschema.StringAttribute{Computed: true}, "data": dataschema.SingleNestedAttribute{
					Attributes: map[string]dataschema.Attribute{
						"size":                    dataschema.StringAttribute{Computed: true},
						"auto_upgrade_disabled":   dataschema.BoolAttribute{Computed: true},
						"target_version":          dataschema.StringAttribute{Computed: true},
						"kustomization":           dataschema.StringAttribute{Computed: true},
						"remote_argocd":           dataschema.StringAttribute{Computed: true},
						"akuity_managed":          dataschema.BoolAttribute{Computed: true},
						"argocd_namespace":        dataschema.StringAttribute{Computed: true},
						"self_managed_argocd_url": dataschema.StringAttribute{Computed: true},
					}, Computed: true,
				},
			}, Computed: true}, "kube_config": dataschema.SingleNestedAttribute{
				Attributes: map[string]dataschema.Attribute{
					"host":     dataschema.StringAttribute{Optional: true},
					"username": dataschema.StringAttribute{Optional: true}, "password": dataschema.StringAttribute{
						Optional: true, Sensitive: true,
					}, "insecure": dataschema.BoolAttribute{Optional: true}, "client_certificate": dataschema.StringAttribute{
						Optional: true,
					}, "client_key": dataschema.StringAttribute{Optional: true, Sensitive: true},
					"cluster_ca_certificate": dataschema.StringAttribute{Optional: true},
					"config_path":            dataschema.StringAttribute{Optional: true}, "config_paths": dataschema.ListAttribute{
						Optional: true, ElementType: types.StringType,
					}, "config_context": dataschema.StringAttribute{Optional: true},
					"config_context_auth_info": dataschema.StringAttribute{Optional: true},
					"config_context_cluster":   dataschema.StringAttribute{Optional: true},
					"token":                    dataschema.StringAttribute{Optional: true, Sensitive: true},
					"proxy_url":                dataschema.StringAttribute{Optional: true}, "exec": dataschema.SingleNestedAttribute{
						Attributes: map[string]dataschema.Attribute{"api_version": dataschema.StringAttribute{
							Computed: true,
						}, "command": dataschema.StringAttribute{Computed: true}, "args": dataschema.ListAttribute{
							Optional: true, ElementType: types.StringType,
						}, "env": dataschema.MapAttribute{Optional: true, ElementType: types.StringType}}, Computed: true,
					},
				}, Computed: true,
			}, "remove_agent_resources_on_destroy": dataschema.BoolAttribute{Computed: true},
		},
	}
	KargoAgentsDataSourceSchema dataschema.Schema = dataschema.Schema{
		Attributes: map[string]dataschema.Attribute{
			"id":          dataschema.StringAttribute{Computed: true},
			"instance_id": dataschema.StringAttribute{Required: true}, "agents": dataschema.ListNestedAttribute{
				NestedObject: dataschema.NestedAttributeObject{Attributes: map[string]dataschema.Attribute{
					"id": dataschema.StringAttribute{Computed: true}, "instance_id": dataschema.StringAttribute{
						Required: true,
					}, "workspace": dataschema.StringAttribute{Computed: true},
					"name": dataschema.StringAttribute{Required: true}, "namespace": dataschema.StringAttribute{
						Computed: true,
					}, "labels": dataschema.MapAttribute{Computed: true, ElementType: types.StringType},
					"annotations": dataschema.MapAttribute{Computed: true, ElementType: types.StringType},
					"spec": dataschema.SingleNestedAttribute{Attributes: map[string]dataschema.Attribute{
						"description": dataschema.StringAttribute{Computed: true}, "data": dataschema.SingleNestedAttribute{
							Attributes: map[string]dataschema.Attribute{
								"size":                    dataschema.StringAttribute{Computed: true},
								"auto_upgrade_disabled":   dataschema.BoolAttribute{Computed: true},
								"target_version":          dataschema.StringAttribute{Computed: true},
								"kustomization":           dataschema.StringAttribute{Computed: true},
								"remote_argocd":           dataschema.StringAttribute{Computed: true},
								"akuity_managed":          dataschema.BoolAttribute{Computed: true},
								"argocd_namespace":        dataschema.StringAttribute{Computed: true},
								"self_managed_argocd_url": dataschema.StringAttribute{Computed: true},
							}, Computed: true,
						},
					}, Computed: true}, "kube_config": dataschema.SingleNestedAttribute{
						Attributes: map[string]dataschema.Attribute{
							"host":     dataschema.StringAttribute{Optional: true},
							"username": dataschema.StringAttribute{Optional: true}, "password": dataschema.StringAttribute{
								Optional: true, Sensitive: true,
							}, "insecure": dataschema.BoolAttribute{Optional: true}, "client_certificate": dataschema.StringAttribute{
								Optional: true,
							}, "client_key": dataschema.StringAttribute{Optional: true, Sensitive: true},
							"cluster_ca_certificate": dataschema.StringAttribute{Optional: true},
							"config_path":            dataschema.StringAttribute{Optional: true}, "config_paths": dataschema.ListAttribute{
								Optional: true, ElementType: types.StringType,
							}, "config_context": dataschema.StringAttribute{Optional: true},
							"config_context_auth_info": dataschema.StringAttribute{Optional: true},
							"config_context_cluster":   dataschema.StringAttribute{Optional: true},
							"token":                    dataschema.StringAttribute{Optional: true, Sensitive: true},
							"proxy_url":                dataschema.StringAttribute{Optional: true}, "exec": dataschema.SingleNestedAttribute{
								Attributes: map[string]dataschema.Attribute{"api_version": dataschema.StringAttribute{
									Computed: true,
								}, "command": dataschema.StringAttribute{Computed: true}, "args": dataschema.ListAttribute{
									Optional: true, ElementType: types.StringType,
								}, "env": dataschema.MapAttribute{Optional: true, ElementType: types.StringType}}, Computed: true,
							},
						}, Computed: true,
					}, "remove_agent_resources_on_destroy": dataschema.BoolAttribute{Computed: true},
				}}, Computed: true,
			},
		},
	}
)
