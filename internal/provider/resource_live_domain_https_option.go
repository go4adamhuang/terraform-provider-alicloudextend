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

var _ resource.Resource = &LiveDomainHttpsOptionResource{}
var _ resource.ResourceWithImportState = &LiveDomainHttpsOptionResource{}

type LiveDomainHttpsOptionResource struct{ client *ClientConfig }

func NewLiveDomainHttpsOptionResource() resource.Resource {
	return &LiveDomainHttpsOptionResource{}
}

type LiveDomainHttpsOptionModel struct {
	DomainName types.String `tfsdk:"domain_name"`
	Http2      types.String `tfsdk:"http2"`
}

func (r *LiveDomainHttpsOptionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_live_domain_https_option"
}

func (r *LiveDomainHttpsOptionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages the HTTPS options (HTTP/2) for an ApsaraVideo Live domain. " +
			"Requires HTTPS to be enabled via alicloudextend_live_domain_certificate first.",
		Attributes: map[string]schema.Attribute{
			"domain_name": schema.StringAttribute{
				Required:    true,
				Description: "The live domain name.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"http2": schema.StringAttribute{
				Required:    true,
				Description: "Whether to enable HTTP/2. Valid values: on, off.",
			},
		},
	}
}

func (r *LiveDomainHttpsOptionResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*ClientConfig)
}

func (r *LiveDomainHttpsOptionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan LiveDomainHttpsOptionModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	live, err := newLiveClientFromConfig(r.client)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create Live client", err.Error())
		return
	}

	if err := applyHttpsOption(live, &plan); err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to set HTTPS option for domain %q", plan.DomainName.ValueString()), err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *LiveDomainHttpsOptionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state LiveDomainHttpsOptionModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	live, err := newLiveClientFromConfig(r.client)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create Live client", err.Error())
		return
	}

	args, err := describeFunctionArgs(live, state.DomainName.ValueString(), "https_option")
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to read HTTPS option for domain %q", state.DomainName.ValueString()), err.Error())
		return
	}
	if args == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	// Prefer live_http2 (Live-specific), fall back to http2.
	if v, ok := args["live_http2"]; ok {
		state.Http2 = types.StringValue(v)
	} else if v, ok := args["http2"]; ok {
		state.Http2 = types.StringValue(v)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *LiveDomainHttpsOptionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan LiveDomainHttpsOptionModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	live, err := newLiveClientFromConfig(r.client)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create Live client", err.Error())
		return
	}

	if err := applyHttpsOption(live, &plan); err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to update HTTPS option for domain %q", plan.DomainName.ValueString()), err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *LiveDomainHttpsOptionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state LiveDomainHttpsOptionModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	live, err := newLiveClientFromConfig(r.client)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create Live client", err.Error())
		return
	}

	if err := batchDeleteConfig(live, state.DomainName.ValueString(), "https_option"); err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to delete HTTPS option for domain %q", state.DomainName.ValueString()), err.Error())
	}
}

func (r *LiveDomainHttpsOptionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("domain_name"), req, resp)
}

func applyHttpsOption(live *liveclient.Client, m *LiveDomainHttpsOptionModel) error {
	// Both http2 and live_http2 are set — live_http2 is the Live-specific arg,
	// http2 is the generic CDN arg. Setting both ensures compatibility.
	return batchSetConfig(live, m.DomainName.ValueString(), "https_option", map[string]string{
		"http2":      m.Http2.ValueString(),
		"live_http2": m.Http2.ValueString(),
	})
}
