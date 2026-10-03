package akp

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	tftypes "github.com/hashicorp/terraform-plugin-framework/types"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"

	serviceaccountv1 "github.com/akuity/api-client-go/pkg/api/gen/serviceaccount/v1"
	"github.com/akuity/terraform-provider-akp/akp/types"
)

func NewAkpOIDCIssuerResource() resource.Resource {
	return &GenericResource[types.OIDCIssuer]{
		TypeNameSuffix:  "oidc_issuer",
		SchemaFunc:      oidcIssuerSchema,
		CreateFunc:      oidcIssuerCreate,
		ReadFunc:        oidcIssuerRead,
		UpdateFunc:      oidcIssuerUpdate,
		DeleteFunc:      oidcIssuerDelete,
		ImportStateFunc: importSplitID("id"),
	}
}

func oidcIssuerCreate(ctx context.Context, cli *AkpCli, _ *diag.Diagnostics, plan *types.OIDCIssuer) (*types.OIDCIssuer, error) {
	spec, err := oidcIssuerSpec(plan)
	if err != nil {
		return nil, err
	}
	resp, err := retryWithBackoff(ctx, func(ctx context.Context) (*serviceaccountv1.CreateOIDCIssuerResponse, error) {
		return cli.ServiceAccountCli.CreateOIDCIssuer(ctx, &serviceaccountv1.CreateOIDCIssuerRequest{
			OrganizationId: cli.OrgId,
			Spec:           spec,
		})
	}, "CreateOIDCIssuer")
	if err != nil {
		return nil, fmt.Errorf("unable to create OIDC issuer: %w", err)
	}
	return plan, applyOIDCIssuerResponse(plan, resp.GetIssuer())
}

func oidcIssuerRead(ctx context.Context, cli *AkpCli, _ *diag.Diagnostics, data *types.OIDCIssuer) error {
	resp, err := retryWithBackoff(ctx, func(ctx context.Context) (*serviceaccountv1.GetOIDCIssuerResponse, error) {
		return cli.ServiceAccountCli.GetOIDCIssuer(ctx, &serviceaccountv1.GetOIDCIssuerRequest{
			OrganizationId: cli.OrgId,
			Id:             data.ID.ValueString(),
		})
	}, "GetOIDCIssuer")
	if err != nil {
		return err
	}
	return applyOIDCIssuerResponse(data, resp.GetIssuer())
}

// oidcIssuerUpdate sends the complete desired state: the server replaces the
// issuer wholesale, and name is RequiresReplace so it always matches state.
func oidcIssuerUpdate(ctx context.Context, cli *AkpCli, _ *diag.Diagnostics, plan *types.OIDCIssuer) (*types.OIDCIssuer, error) {
	spec, err := oidcIssuerSpec(plan)
	if err != nil {
		return nil, err
	}
	resp, err := retryWithBackoff(ctx, func(ctx context.Context) (*serviceaccountv1.UpdateOIDCIssuerResponse, error) {
		return cli.ServiceAccountCli.UpdateOIDCIssuer(ctx, &serviceaccountv1.UpdateOIDCIssuerRequest{
			OrganizationId: cli.OrgId,
			Id:             plan.ID.ValueString(),
			Spec:           spec,
		})
	}, "UpdateOIDCIssuer")
	if err != nil {
		return nil, fmt.Errorf("unable to update OIDC issuer: %w", err)
	}
	return plan, applyOIDCIssuerResponse(plan, resp.GetIssuer())
}

func oidcIssuerDelete(ctx context.Context, cli *AkpCli, _ *diag.Diagnostics, state *types.OIDCIssuer) error {
	_, err := retryWithBackoff(ctx, func(ctx context.Context) (*serviceaccountv1.DeleteOIDCIssuerResponse, error) {
		resp, err := cli.ServiceAccountCli.DeleteOIDCIssuer(ctx, &serviceaccountv1.DeleteOIDCIssuerRequest{
			OrganizationId: cli.OrgId,
			Id:             state.ID.ValueString(),
		})
		if isGoneErr(err) {
			return resp, nil
		}
		return resp, err
	}, "DeleteOIDCIssuer")
	if err != nil {
		return fmt.Errorf("unable to delete OIDC issuer: %w", err)
	}
	return nil
}

// oidcIssuerSpec builds the wire spec, checking that the attribute the chosen
// key source needs is present so the mistake reads as a config error rather
// than a server rejection.
func oidcIssuerSpec(plan *types.OIDCIssuer) (*serviceaccountv1.OIDCIssuerSpec, error) {
	spec := &serviceaccountv1.OIDCIssuerSpec{
		Name:            plan.Name.ValueString(),
		IssuerUrl:       plan.IssuerURL.ValueString(),
		Audiences:       stringSliceFromTF(plan.Audiences),
		TokenTtlSeconds: plan.TokenTTLSeconds.ValueInt64(),
	}
	source := plan.KeySource.ValueString()
	if source != oidcKeySourceJWKSURI && plan.JWKSURI.ValueString() != "" {
		return nil, fmt.Errorf("jwks_uri must be unset when key_source is %q", source)
	}
	if source != oidcKeySourceStaticJWKS && plan.StaticJWKS.ValueString() != "" {
		return nil, fmt.Errorf("static_jwks must be unset when key_source is %q", source)
	}
	switch source {
	case oidcKeySourceDiscovery:
		spec.KeySource = &serviceaccountv1.OIDCIssuerKeySource{
			Source: &serviceaccountv1.OIDCIssuerKeySource_Discovery_{Discovery: &serviceaccountv1.OIDCIssuerKeySource_Discovery{}},
		}
	case oidcKeySourceJWKSURI:
		uri := plan.JWKSURI.ValueString()
		if uri == "" {
			return nil, fmt.Errorf("jwks_uri is required when key_source is %q", oidcKeySourceJWKSURI)
		}
		spec.KeySource = &serviceaccountv1.OIDCIssuerKeySource{
			Source: &serviceaccountv1.OIDCIssuerKeySource_JwksUri{JwksUri: uri},
		}
	case oidcKeySourceStaticJWKS:
		raw := plan.StaticJWKS.ValueString()
		if raw == "" {
			return nil, fmt.Errorf("static_jwks is required when key_source is %q", oidcKeySourceStaticJWKS)
		}
		var doc map[string]any
		if err := json.Unmarshal([]byte(raw), &doc); err != nil {
			return nil, fmt.Errorf("static_jwks must be a JSON object: %w", err)
		}
		st, err := structpb.NewStruct(doc)
		if err != nil {
			return nil, fmt.Errorf("static_jwks: %w", err)
		}
		spec.KeySource = &serviceaccountv1.OIDCIssuerKeySource{
			Source: &serviceaccountv1.OIDCIssuerKeySource_StaticJwks{StaticJwks: st},
		}
	default:
		return nil, fmt.Errorf("key_source must be one of %q, %q, %q", oidcKeySourceDiscovery, oidcKeySourceJWKSURI, oidcKeySourceStaticJWKS)
	}
	return spec, nil
}

func applyOIDCIssuerResponse(data *types.OIDCIssuer, iss *serviceaccountv1.OIDCIssuer) error {
	if iss == nil {
		return nil
	}
	data.ID = tftypes.StringValue(iss.GetId())
	data.Name = tftypes.StringValue(iss.GetName())
	data.IssuerURL = tftypes.StringValue(iss.GetIssuerUrl())
	data.DefaultAudience = tftypes.StringValue(iss.GetDefaultAudience())
	data.TokenTTLSeconds = tftypes.Int64Value(iss.GetTokenTtlSeconds())
	// An empty response keeps a null config null (see applyStringList).
	data.Audiences = applyStringList(data.Audiences, iss.GetAudiences())
	switch source := iss.GetKeySource().GetSource().(type) {
	case *serviceaccountv1.OIDCIssuerKeySource_Discovery_:
		data.KeySource = tftypes.StringValue(oidcKeySourceDiscovery)
		data.JWKSURI = tftypes.StringNull()
		data.StaticJWKS = tftypes.StringNull()
	case *serviceaccountv1.OIDCIssuerKeySource_JwksUri:
		data.KeySource = tftypes.StringValue(oidcKeySourceJWKSURI)
		data.JWKSURI = tftypes.StringValue(source.JwksUri)
		data.StaticJWKS = tftypes.StringNull()
	case *serviceaccountv1.OIDCIssuerKeySource_StaticJwks:
		data.KeySource = tftypes.StringValue(oidcKeySourceStaticJWKS)
		data.JWKSURI = tftypes.StringNull()
		raw, err := protojson.Marshal(source.StaticJwks)
		if err != nil {
			return fmt.Errorf("decode static_jwks: %w", err)
		}
		data.StaticJWKS = applyJSONString(data.StaticJWKS, string(raw))
	}
	if t := iss.GetKeysUpdatedTimestamp(); t != nil {
		data.KeysUpdatedTime = tftypes.StringValue(t.AsTime().Format(time.RFC3339))
	} else {
		data.KeysUpdatedTime = tftypes.StringValue("")
	}
	if t := iss.GetCreationTimestamp(); t != nil {
		data.CreateTime = tftypes.StringValue(t.AsTime().Format(time.RFC3339))
	}
	return nil
}

// applyJSONString keeps the operator's JSON text when it means the same as
// the server's re-encoding, so formatting and key order never diff; anything
// else takes the server value.
func applyJSONString(current tftypes.String, server string) tftypes.String {
	if current.IsNull() || current.IsUnknown() {
		return tftypes.StringValue(server)
	}
	var mine, theirs any
	if json.Unmarshal([]byte(current.ValueString()), &mine) == nil &&
		json.Unmarshal([]byte(server), &theirs) == nil &&
		reflect.DeepEqual(mine, theirs) {
		return current
	}
	return tftypes.StringValue(server)
}
