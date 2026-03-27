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

var _ resource.Resource = &LiveDomainIpv6Resource{}
var _ resource.ResourceWithImportState = &LiveDomainIpv6Resource{}

type LiveDomainIpv6Resource struct{ client *ClientConfig }

func NewLiveDomainIpv6Resource() resource.Resource {
	return &LiveDomainIpv6Resource{}
}

type LiveDomainIpv6Model struct {
	DomainName types.String `tfsdk:"domain_name"`
	Switch     types.String `tfsdk:"switch"`
}

func (r *LiveDomainIpv6Resource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_live_domain_ipv6"
}

func (r *LiveDomainIpv6Resource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages the IPv6 configuration for an ApsaraVideo Live domain.",
		Attributes: map[string]schema.Attribute{
			"domain_name": schema.StringAttribute{
				Required:    true,
				Description: "The live domain name.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"switch": schema.StringAttribute{
				Required:    true,
				Description: "Whether to enable IPv6. Valid values: on, off.",
			},
		},
	}
}

func (r *LiveDomainIpv6Resource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*ClientConfig)
}

func (r *LiveDomainIpv6Resource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan LiveDomainIpv6Model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	live, err := newLiveClientFromConfig(r.client)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create Live client", err.Error())
		return
	}

	if err := applyIpv6(live, &plan); err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to set IPv6 for domain %q", plan.DomainName.ValueString()), err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *LiveDomainIpv6Resource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state LiveDomainIpv6Model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	live, err := newLiveClientFromConfig(r.client)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create Live client", err.Error())
		return
	}

	args, err := describeFunctionArgs(live, state.DomainName.ValueString(), "ipv6")
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to read IPv6 config for domain %q", state.DomainName.ValueString()), err.Error())
		return
	}
	if args == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	if v, ok := args["switch"]; ok {
		state.Switch = types.StringValue(v)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *LiveDomainIpv6Resource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan LiveDomainIpv6Model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	live, err := newLiveClientFromConfig(r.client)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create Live client", err.Error())
		return
	}

	if err := applyIpv6(live, &plan); err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to update IPv6 for domain %q", plan.DomainName.ValueString()), err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *LiveDomainIpv6Resource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state LiveDomainIpv6Model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	live, err := newLiveClientFromConfig(r.client)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create Live client", err.Error())
		return
	}

	if err := batchDeleteConfig(live, state.DomainName.ValueString(), "ipv6"); err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to delete IPv6 config for domain %q", state.DomainName.ValueString()), err.Error())
	}
}

func (r *LiveDomainIpv6Resource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("domain_name"), req, resp)
}

func applyIpv6(live *liveclient.Client, m *LiveDomainIpv6Model) error {
	return batchSetConfig(live, m.DomainName.ValueString(), "ipv6", map[string]string{
		"switch": m.Switch.ValueString(),
	})
}
