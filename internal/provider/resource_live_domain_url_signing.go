package provider

import (
	"context"
	"fmt"

	liveclient "github.com/alibabacloud-go/live-20161101/v2/client"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &LiveDomainUrlSigningResource{}
var _ resource.ResourceWithImportState = &LiveDomainUrlSigningResource{}

type LiveDomainUrlSigningResource struct{ client *ClientConfig }

func NewLiveDomainUrlSigningResource() resource.Resource {
	return &LiveDomainUrlSigningResource{}
}

type LiveDomainUrlSigningModel struct {
	DomainName types.String `tfsdk:"domain_name"`
	Enabled    types.Bool   `tfsdk:"enabled"`
}

func (r *LiveDomainUrlSigningResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_live_domain_url_signing"
}

func (r *LiveDomainUrlSigningResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages the URL signing (URL authentication) settings for an ApsaraVideo Live domain. " +
			"Set enabled = false to disable URL signing (auth_type = no_auth). " +
			"Applies to both ingest and play domains.",
		Attributes: map[string]schema.Attribute{
			"domain_name": schema.StringAttribute{
				Required:    true,
				Description: "The live domain name.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"enabled": schema.BoolAttribute{
				Required:    true,
				Description: "Whether URL signing is enabled. Set to false to disable URL signing (auth_type = no_auth).",
			},
		},
	}
}

func (r *LiveDomainUrlSigningResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*ClientConfig)
}

func (r *LiveDomainUrlSigningResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan LiveDomainUrlSigningModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	live, err := newLiveClientFromConfig(r.client)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create Live client", err.Error())
		return
	}

	if err := applyUrlSigning(live, &plan); err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to set URL signing for domain %q", plan.DomainName.ValueString()), err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *LiveDomainUrlSigningResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state LiveDomainUrlSigningModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	live, err := newLiveClientFromConfig(r.client)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create Live client", err.Error())
		return
	}

	args, err := describeFunctionArgs(live, state.DomainName.ValueString(), "aliauth")
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to read URL signing config for domain %q", state.DomainName.ValueString()), err.Error())
		return
	}

	if args == nil {
		state.Enabled = types.BoolValue(false)
	} else if authType, ok := args["auth_type"]; ok {
		state.Enabled = types.BoolValue(authType != "no_auth" && authType != "")
	} else {
		state.Enabled = types.BoolValue(false)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *LiveDomainUrlSigningResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan LiveDomainUrlSigningModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	live, err := newLiveClientFromConfig(r.client)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create Live client", err.Error())
		return
	}

	if err := applyUrlSigning(live, &plan); err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to update URL signing for domain %q", plan.DomainName.ValueString()), err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *LiveDomainUrlSigningResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state LiveDomainUrlSigningModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	live, err := newLiveClientFromConfig(r.client)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create Live client", err.Error())
		return
	}

	if err := batchDeleteConfig(live, state.DomainName.ValueString(), "aliauth"); err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to delete URL signing config for domain %q", state.DomainName.ValueString()), err.Error())
	}
}

func (r *LiveDomainUrlSigningResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("domain_name"), req, resp)
}

func applyUrlSigning(live *liveclient.Client, m *LiveDomainUrlSigningModel) error {
	authType := "no_auth"
	if m.Enabled.ValueBool() {
		authType = "type_a"
	}
	return batchSetConfig(live, m.DomainName.ValueString(), "aliauth", map[string]string{
		"auth_type": authType,
	})
}
