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

var _ resource.Resource = &LiveDomainAccessControlResource{}
var _ resource.ResourceWithImportState = &LiveDomainAccessControlResource{}

type LiveDomainAccessControlResource struct{ client *ClientConfig }

func NewLiveDomainAccessControlResource() resource.Resource {
	return &LiveDomainAccessControlResource{}
}

type LiveDomainAccessControlModel struct {
	DomainName types.String `tfsdk:"domain_name"`
	RtmpBlock  types.String `tfsdk:"rtmp_block"`
	HlsBlock   types.String `tfsdk:"hls_block"`
}

func (r *LiveDomainAccessControlResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_live_domain_access_control"
}

func (r *LiveDomainAccessControlResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages the protocol blocking (access control) settings for an ApsaraVideo Live domain. " +
			"Allows blocking RTMP and/or HLS playback protocols.",
		Attributes: map[string]schema.Attribute{
			"domain_name": schema.StringAttribute{
				Required:    true,
				Description: "The live domain name.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"rtmp_block": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether to block RTMP playback. Valid values: on, off. Defaults to off.",
			},
			"hls_block": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether to block HLS playback. Valid values: on, off. Defaults to off.",
			},
		},
	}
}

func (r *LiveDomainAccessControlResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*ClientConfig)
}

func (r *LiveDomainAccessControlResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan LiveDomainAccessControlModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	live, err := newLiveClientFromConfig(r.client)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create Live client", err.Error())
		return
	}

	if err := applyAccessControl(live, &plan); err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to set access control for domain %q", plan.DomainName.ValueString()), err.Error())
		return
	}

	readAccessControl(live, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *LiveDomainAccessControlResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state LiveDomainAccessControlModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	live, err := newLiveClientFromConfig(r.client)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create Live client", err.Error())
		return
	}

	readAccessControl(live, &state, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *LiveDomainAccessControlResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan LiveDomainAccessControlModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	live, err := newLiveClientFromConfig(r.client)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create Live client", err.Error())
		return
	}

	if err := applyAccessControl(live, &plan); err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to update access control for domain %q", plan.DomainName.ValueString()), err.Error())
		return
	}

	readAccessControl(live, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *LiveDomainAccessControlResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state LiveDomainAccessControlModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	live, err := newLiveClientFromConfig(r.client)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create Live client", err.Error())
		return
	}

	if err := batchDeleteConfig(live, state.DomainName.ValueString(), "alilive"); err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to delete access control config for domain %q", state.DomainName.ValueString()), err.Error())
	}
}

func (r *LiveDomainAccessControlResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("domain_name"), req, resp)
}

func applyAccessControl(live *liveclient.Client, m *LiveDomainAccessControlModel) error {
	rtmpBlock := "off"
	if !m.RtmpBlock.IsNull() && !m.RtmpBlock.IsUnknown() {
		rtmpBlock = m.RtmpBlock.ValueString()
	}
	hlsBlock := "off"
	if !m.HlsBlock.IsNull() && !m.HlsBlock.IsUnknown() {
		hlsBlock = m.HlsBlock.ValueString()
	}
	return batchSetConfig(live, m.DomainName.ValueString(), "alilive", map[string]string{
		"enable":           "on",
		"live_forbid_rtmp": rtmpBlock,
		"live_forbid_hls":  hlsBlock,
	})
}

func readAccessControl(live *liveclient.Client, m *LiveDomainAccessControlModel, diags interface {
	AddError(summary, detail string)
}) {
	args, err := describeFunctionArgs(live, m.DomainName.ValueString(), "alilive")
	if err != nil {
		diags.AddError(fmt.Sprintf("Failed to read access control config for domain %q", m.DomainName.ValueString()), err.Error())
		return
	}
	if args == nil {
		m.RtmpBlock = types.StringValue("off")
		m.HlsBlock = types.StringValue("off")
		return
	}
	if v, ok := args["live_forbid_rtmp"]; ok {
		m.RtmpBlock = types.StringValue(v)
	}
	if v, ok := args["live_forbid_hls"]; ok {
		m.HlsBlock = types.StringValue(v)
	}
}
