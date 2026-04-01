package akp

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                     = &GenericResource[any]{}
	_ resource.ResourceWithImportState      = &GenericResource[any]{}
	_ resource.ResourceWithConfigValidators = &GenericResource[any]{}
)

type GenericResource[Plan any] struct {
	BaseResource
	TypeNameSuffix       string
	SchemaFunc           func() schema.Schema
	CreateFunc           func(ctx context.Context, cli *AkpCli, diags *diag.Diagnostics, plan *Plan) (*Plan, error)
	ReadFunc             func(ctx context.Context, cli *AkpCli, diags *diag.Diagnostics, data *Plan) error
	UpdateFunc           func(ctx context.Context, cli *AkpCli, diags *diag.Diagnostics, plan *Plan) (*Plan, error)
	DeleteFunc           func(ctx context.Context, cli *AkpCli, diags *diag.Diagnostics, state *Plan) error
	ImportStateFunc      func(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse)
	ConfigValidatorsFunc func() []resource.ConfigValidator
}

func (r *GenericResource[Plan]) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_" + r.TypeNameSuffix
}

func (r *GenericResource[Plan]) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = r.SchemaFunc()
}

func (r *GenericResource[Plan]) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Debug(ctx, fmt.Sprintf("Creating %s", r.TypeNameSuffix))
	var plan Plan
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx = r.AuthCtx(ctx)
	result, err := r.CreateFunc(ctx, r.akpCli, &resp.Diagnostics, &plan)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
	}
	if result != nil {
		resp.Diagnostics.Append(resp.State.Set(ctx, result)...)
	}
}

func (r *GenericResource[Plan]) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Debug(ctx, fmt.Sprintf("Reading %s", r.TypeNameSuffix))
	var data Plan
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx = r.AuthCtx(ctx)
	if err := r.ReadFunc(ctx, r.akpCli, &resp.Diagnostics, &data); err != nil {
		handleReadResourceError(ctx, resp, err)
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *GenericResource[Plan]) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Debug(ctx, fmt.Sprintf("Updating %s", r.TypeNameSuffix))
	var plan Plan
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx = r.AuthCtx(ctx)
	result, err := r.UpdateFunc(ctx, r.akpCli, &resp.Diagnostics, &plan)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
	}
	if result != nil {
		resp.Diagnostics.Append(resp.State.Set(ctx, result)...)
	}
}

func (r *GenericResource[Plan]) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Debug(ctx, fmt.Sprintf("Deleting %s", r.TypeNameSuffix))
	var state Plan
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx = r.AuthCtx(ctx)
	if err := r.DeleteFunc(ctx, r.akpCli, &resp.Diagnostics, &state); err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
	}
}

func (r *GenericResource[Plan]) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if r.ImportStateFunc != nil {
		r.ImportStateFunc(ctx, req, resp)
		return
	}
	resp.Diagnostics.AddError("Import Not Supported", fmt.Sprintf("Import is not supported for %s", r.TypeNameSuffix))
}

func (r *GenericResource[Plan]) ConfigValidators(_ context.Context) []resource.ConfigValidator {
	if r.ConfigValidatorsFunc != nil {
		return r.ConfigValidatorsFunc()
	}
	return nil
}
