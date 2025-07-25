// This is an auto-generated file. DO NOT EDIT
/*
Copyright 2025 Akuity, Inc.
*/

package types

import (
	dataschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	resourceschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	ClusterResourceSchema resourceschema.Schema = resourceschema.Schema{
		Attributes: map[string]resourceschema.Attribute{
			"id": resourceschema.StringAttribute{Computed: true},
			"instance_id": resourceschema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			}, "name": resourceschema.StringAttribute{Required: true, PlanModifiers: []planmodifier.String{
				stringplanmodifier.RequiresReplace(),
			}}, "namespace": resourceschema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
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
				}, "namespace_scoped": resourceschema.BoolAttribute{
					Optional: true, Computed: true,
					PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
				}, "data": resourceschema.SingleNestedAttribute{Attributes: map[string]resourceschema.Attribute{
					"size": resourceschema.StringAttribute{Required: true}, "auto_upgrade_disabled": resourceschema.BoolAttribute{
						Optional: true, Computed: true, PlanModifiers: []planmodifier.Bool{
							boolplanmodifier.UseStateForUnknown(),
						},
					}, "kustomization": resourceschema.StringAttribute{
						Optional: true, Computed: true,
						PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
					}, "app_replication": resourceschema.BoolAttribute{
						Optional: true, Computed: true,
						PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
					}, "target_version": resourceschema.StringAttribute{
						Optional: true, Computed: true,
						PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
					}, "redis_tunneling": resourceschema.BoolAttribute{
						Optional: true, Computed: true,
						PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
					}, "datadog_annotations_enabled": resourceschema.BoolAttribute{
						Optional: true, Computed: true,
						PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
					}, "eks_addon_enabled": resourceschema.BoolAttribute{
						Optional: true, Computed: true,
						PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
					}, "managed_cluster_config": resourceschema.SingleNestedAttribute{
						Attributes: map[string]resourceschema.Attribute{"secret_name": resourceschema.StringAttribute{
							Optional: true, Computed: true, PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						}, "secret_key": resourceschema.StringAttribute{
							Optional: true, Computed: true,
							PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
						}}, Optional: true, PlanModifiers: []planmodifier.Object{objectplanmodifier.UseStateForUnknown()},
					}, "multi_cluster_k8s_dashboard_enabled": resourceschema.BoolAttribute{
						Optional: true,
						Computed: true, PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
					}, "custom_agent_size_config": resourceschema.SingleNestedAttribute{
						Attributes: map[string]resourceschema.Attribute{"application_controller": resourceschema.SingleNestedAttribute{
							Attributes: map[string]resourceschema.Attribute{"memory": resourceschema.StringAttribute{
								Optional: true, Computed: true, PlanModifiers: []planmodifier.String{
									stringplanmodifier.UseStateForUnknown(),
								},
							}, "cpu": resourceschema.StringAttribute{
								Optional: true, Computed: true,
								PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
							}}, Optional: true, PlanModifiers: []planmodifier.Object{objectplanmodifier.UseStateForUnknown()},
						}, "repo_server": resourceschema.SingleNestedAttribute{Attributes: map[string]resourceschema.Attribute{
							"memory": resourceschema.StringAttribute{
								Optional: true, Computed: true,
								PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
							}, "cpu": resourceschema.StringAttribute{
								Optional: true, Computed: true,
								PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
							}, "replicas": resourceschema.Int64Attribute{
								Optional: true, Computed: true,
								PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
							},
						}, Optional: true, PlanModifiers: []planmodifier.Object{objectplanmodifier.UseStateForUnknown()}}},
						Optional: true, PlanModifiers: []planmodifier.Object{objectplanmodifier.UseStateForUnknown()},
					}, "autoscaler_config": resourceschema.SingleNestedAttribute{
						Attributes: map[string]resourceschema.Attribute{"application_controller": resourceschema.SingleNestedAttribute{
							Attributes: map[string]resourceschema.Attribute{"resource_minimum": resourceschema.SingleNestedAttribute{
								Attributes: map[string]resourceschema.Attribute{"memory": resourceschema.StringAttribute{
									Optional: true, Computed: true, PlanModifiers: []planmodifier.String{
										stringplanmodifier.UseStateForUnknown(),
									},
								}, "cpu": resourceschema.StringAttribute{
									Optional: true, Computed: true,
									PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
								}}, Optional: true, PlanModifiers: []planmodifier.Object{objectplanmodifier.UseStateForUnknown()},
							}, "resource_maximum": resourceschema.SingleNestedAttribute{
								Attributes: map[string]resourceschema.Attribute{"memory": resourceschema.StringAttribute{
									Optional: true, Computed: true, PlanModifiers: []planmodifier.String{
										stringplanmodifier.UseStateForUnknown(),
									},
								}, "cpu": resourceschema.StringAttribute{
									Optional: true, Computed: true,
									PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
								}}, Optional: true, PlanModifiers: []planmodifier.Object{objectplanmodifier.UseStateForUnknown()},
							}}, Optional: true,
						}, "repo_server": resourceschema.SingleNestedAttribute{Attributes: map[string]resourceschema.Attribute{
							"resource_minimum": resourceschema.SingleNestedAttribute{Attributes: map[string]resourceschema.Attribute{
								"memory": resourceschema.StringAttribute{
									Optional: true, Computed: true,
									PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
								}, "cpu": resourceschema.StringAttribute{
									Optional: true, Computed: true,
									PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
								},
							}, Optional: true, PlanModifiers: []planmodifier.Object{objectplanmodifier.UseStateForUnknown()}},
							"resource_maximum": resourceschema.SingleNestedAttribute{Attributes: map[string]resourceschema.Attribute{
								"memory": resourceschema.StringAttribute{
									Optional: true, Computed: true,
									PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
								}, "cpu": resourceschema.StringAttribute{
									Optional: true, Computed: true,
									PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
								},
							}, Optional: true, PlanModifiers: []planmodifier.Object{objectplanmodifier.UseStateForUnknown()}},
							"replicas_maximum": resourceschema.Int64Attribute{
								Optional: true, Computed: true,
								PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
							}, "replicas_minimum": resourceschema.Int64Attribute{
								Optional: true, Computed: true,
								PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
							},
						}, Optional: true}}, Optional: true, Computed: true, PlanModifiers: []planmodifier.Object{
							objectplanmodifier.UseStateForUnknown(),
						},
					}, "project": resourceschema.StringAttribute{
						Optional: true, Computed: true,
						PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
					}, "compatibility": resourceschema.SingleNestedAttribute{Attributes: map[string]resourceschema.Attribute{
						"ipv6only": resourceschema.BoolAttribute{
							Optional: true, Computed: true,
							PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
						},
					}, Optional: true, PlanModifiers: []planmodifier.Object{objectplanmodifier.UseStateForUnknown()}},
					"argocd_notifications_settings": resourceschema.SingleNestedAttribute{
						Attributes: map[string]resourceschema.Attribute{"in_cluster_settings": resourceschema.BoolAttribute{
							Optional: true, Computed: true, PlanModifiers: []planmodifier.Bool{
								boolplanmodifier.UseStateForUnknown(),
							},
						}}, Optional: true, PlanModifiers: []planmodifier.Object{objectplanmodifier.UseStateForUnknown()},
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
	ClusterDataSourceSchema dataschema.Schema = dataschema.Schema{
		Attributes: map[string]dataschema.Attribute{
			"id":          dataschema.StringAttribute{Computed: true},
			"instance_id": dataschema.StringAttribute{Required: true}, "name": dataschema.StringAttribute{
				Required: true,
			}, "namespace": dataschema.StringAttribute{Computed: true},
			"labels":      dataschema.MapAttribute{Computed: true, ElementType: types.StringType},
			"annotations": dataschema.MapAttribute{Computed: true, ElementType: types.StringType},
			"spec": dataschema.SingleNestedAttribute{Attributes: map[string]dataschema.Attribute{
				"description": dataschema.StringAttribute{Computed: true}, "namespace_scoped": dataschema.BoolAttribute{
					Computed: true,
				}, "data": dataschema.SingleNestedAttribute{Attributes: map[string]dataschema.Attribute{
					"size": dataschema.StringAttribute{Computed: true}, "auto_upgrade_disabled": dataschema.BoolAttribute{
						Computed: true,
					}, "kustomization": dataschema.StringAttribute{Computed: true},
					"app_replication":             dataschema.BoolAttribute{Computed: true},
					"target_version":              dataschema.StringAttribute{Computed: true},
					"redis_tunneling":             dataschema.BoolAttribute{Computed: true},
					"datadog_annotations_enabled": dataschema.BoolAttribute{Computed: true},
					"eks_addon_enabled":           dataschema.BoolAttribute{Computed: true},
					"managed_cluster_config": dataschema.SingleNestedAttribute{Attributes: map[string]dataschema.Attribute{
						"secret_name": dataschema.StringAttribute{Computed: true}, "secret_key": dataschema.StringAttribute{
							Computed: true,
						},
					}, Computed: true}, "multi_cluster_k8s_dashboard_enabled": dataschema.BoolAttribute{Computed: true},
					"custom_agent_size_config": dataschema.SingleNestedAttribute{
						Attributes: map[string]dataschema.Attribute{"application_controller": dataschema.SingleNestedAttribute{
							Attributes: map[string]dataschema.Attribute{
								"memory": dataschema.StringAttribute{Computed: true},
								"cpu":    dataschema.StringAttribute{Computed: true},
							}, Computed: true,
						}, "repo_server": dataschema.SingleNestedAttribute{Attributes: map[string]dataschema.Attribute{
							"memory": dataschema.StringAttribute{Computed: true}, "cpu": dataschema.StringAttribute{
								Computed: true,
							}, "replicas": dataschema.Int64Attribute{Computed: true},
						}, Computed: true}}, Computed: true,
					}, "autoscaler_config": dataschema.SingleNestedAttribute{Attributes: map[string]dataschema.Attribute{
						"application_controller": dataschema.SingleNestedAttribute{Attributes: map[string]dataschema.Attribute{
							"resource_minimum": dataschema.SingleNestedAttribute{Attributes: map[string]dataschema.Attribute{
								"memory": dataschema.StringAttribute{Computed: true}, "cpu": dataschema.StringAttribute{
									Computed: true,
								},
							}, Computed: true}, "resource_maximum": dataschema.SingleNestedAttribute{
								Attributes: map[string]dataschema.Attribute{
									"memory": dataschema.StringAttribute{Computed: true},
									"cpu":    dataschema.StringAttribute{Computed: true},
								}, Computed: true,
							},
						}, Computed: true}, "repo_server": dataschema.SingleNestedAttribute{
							Attributes: map[string]dataschema.Attribute{
								"resource_minimum": dataschema.SingleNestedAttribute{
									Attributes: map[string]dataschema.Attribute{
										"memory": dataschema.StringAttribute{Computed: true},
										"cpu":    dataschema.StringAttribute{Computed: true},
									}, Computed: true,
								}, "resource_maximum": dataschema.SingleNestedAttribute{Attributes: map[string]dataschema.Attribute{
									"memory": dataschema.StringAttribute{Computed: true}, "cpu": dataschema.StringAttribute{
										Computed: true,
									},
								}, Computed: true}, "replicas_maximum": dataschema.Int64Attribute{Computed: true},
								"replicas_minimum": dataschema.Int64Attribute{Computed: true},
							}, Computed: true,
						},
					}, Computed: true}, "project": dataschema.StringAttribute{Computed: true},
					"compatibility": dataschema.SingleNestedAttribute{Attributes: map[string]dataschema.Attribute{
						"ipv6only": dataschema.BoolAttribute{Computed: true},
					}, Computed: true}, "argocd_notifications_settings": dataschema.SingleNestedAttribute{
						Attributes: map[string]dataschema.Attribute{"in_cluster_settings": dataschema.BoolAttribute{
							Computed: true,
						}}, Computed: true,
					},
				}, Computed: true},
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
	ClustersDataSourceSchema dataschema.Schema = dataschema.Schema{
		Attributes: map[string]dataschema.Attribute{
			"id":          dataschema.StringAttribute{Computed: true},
			"instance_id": dataschema.StringAttribute{Required: true}, "clusters": dataschema.ListNestedAttribute{
				NestedObject: dataschema.NestedAttributeObject{Attributes: map[string]dataschema.Attribute{
					"id": dataschema.StringAttribute{Computed: true}, "instance_id": dataschema.StringAttribute{
						Required: true,
					}, "name": dataschema.StringAttribute{Required: true}, "namespace": dataschema.StringAttribute{
						Computed: true,
					}, "labels": dataschema.MapAttribute{Computed: true, ElementType: types.StringType},
					"annotations": dataschema.MapAttribute{Computed: true, ElementType: types.StringType},
					"spec": dataschema.SingleNestedAttribute{Attributes: map[string]dataschema.Attribute{
						"description": dataschema.StringAttribute{Computed: true}, "namespace_scoped": dataschema.BoolAttribute{
							Computed: true,
						}, "data": dataschema.SingleNestedAttribute{Attributes: map[string]dataschema.Attribute{
							"size": dataschema.StringAttribute{Computed: true}, "auto_upgrade_disabled": dataschema.BoolAttribute{
								Computed: true,
							}, "kustomization": dataschema.StringAttribute{Computed: true},
							"app_replication":             dataschema.BoolAttribute{Computed: true},
							"target_version":              dataschema.StringAttribute{Computed: true},
							"redis_tunneling":             dataschema.BoolAttribute{Computed: true},
							"datadog_annotations_enabled": dataschema.BoolAttribute{Computed: true},
							"eks_addon_enabled":           dataschema.BoolAttribute{Computed: true},
							"managed_cluster_config": dataschema.SingleNestedAttribute{Attributes: map[string]dataschema.Attribute{
								"secret_name": dataschema.StringAttribute{Computed: true}, "secret_key": dataschema.StringAttribute{
									Computed: true,
								},
							}, Computed: true}, "multi_cluster_k8s_dashboard_enabled": dataschema.BoolAttribute{Computed: true},
							"custom_agent_size_config": dataschema.SingleNestedAttribute{
								Attributes: map[string]dataschema.Attribute{"application_controller": dataschema.SingleNestedAttribute{
									Attributes: map[string]dataschema.Attribute{
										"memory": dataschema.StringAttribute{Computed: true},
										"cpu":    dataschema.StringAttribute{Computed: true},
									}, Computed: true,
								}, "repo_server": dataschema.SingleNestedAttribute{Attributes: map[string]dataschema.Attribute{
									"memory": dataschema.StringAttribute{Computed: true}, "cpu": dataschema.StringAttribute{
										Computed: true,
									}, "replicas": dataschema.Int64Attribute{Computed: true},
								}, Computed: true}}, Computed: true,
							}, "autoscaler_config": dataschema.SingleNestedAttribute{Attributes: map[string]dataschema.Attribute{
								"application_controller": dataschema.SingleNestedAttribute{Attributes: map[string]dataschema.Attribute{
									"resource_minimum": dataschema.SingleNestedAttribute{Attributes: map[string]dataschema.Attribute{
										"memory": dataschema.StringAttribute{Computed: true}, "cpu": dataschema.StringAttribute{
											Computed: true,
										},
									}, Computed: true}, "resource_maximum": dataschema.SingleNestedAttribute{
										Attributes: map[string]dataschema.Attribute{
											"memory": dataschema.StringAttribute{Computed: true},
											"cpu":    dataschema.StringAttribute{Computed: true},
										}, Computed: true,
									},
								}, Computed: true}, "repo_server": dataschema.SingleNestedAttribute{
									Attributes: map[string]dataschema.Attribute{
										"resource_minimum": dataschema.SingleNestedAttribute{
											Attributes: map[string]dataschema.Attribute{
												"memory": dataschema.StringAttribute{Computed: true},
												"cpu":    dataschema.StringAttribute{Computed: true},
											}, Computed: true,
										}, "resource_maximum": dataschema.SingleNestedAttribute{Attributes: map[string]dataschema.Attribute{
											"memory": dataschema.StringAttribute{Computed: true}, "cpu": dataschema.StringAttribute{
												Computed: true,
											},
										}, Computed: true}, "replicas_maximum": dataschema.Int64Attribute{Computed: true},
										"replicas_minimum": dataschema.Int64Attribute{Computed: true},
									}, Computed: true,
								},
							}, Computed: true}, "project": dataschema.StringAttribute{Computed: true},
							"compatibility": dataschema.SingleNestedAttribute{Attributes: map[string]dataschema.Attribute{
								"ipv6only": dataschema.BoolAttribute{Computed: true},
							}, Computed: true}, "argocd_notifications_settings": dataschema.SingleNestedAttribute{
								Attributes: map[string]dataschema.Attribute{"in_cluster_settings": dataschema.BoolAttribute{
									Computed: true,
								}}, Computed: true,
							},
						}, Computed: true},
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
