package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &LiveDomainHttpHeaderResource{}
var _ resource.ResourceWithImportState = &LiveDomainHttpHeaderResource{}

type LiveDomainHttpHeaderResource struct{ client *ClientConfig }

func NewLiveDomainHttpHeaderResource() resource.Resource {
	return &LiveDomainHttpHeaderResource{}
}

type LiveDomainHttpHeaderModel struct {
	DomainName types.String `tfsdk:"domain_name"`
	Key        types.String `tfsdk:"key"`
	Value      types.String `tfsdk:"value"`
}

func (r *LiveDomainHttpHeaderResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_live_domain_http_header"
}

func (r *LiveDomainHttpHeaderResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a single HTTP response header for an ApsaraVideo Live domain. " +
			"Each resource represents one header key-value pair. Multiple resources can be " +
			"created for the same domain to manage different headers independently.",
		Attributes: map[string]schema.Attribute{
			"domain_name": schema.StringAttribute{
				Required:    true,
				Description: "The live domain name.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"key": schema.StringAttribute{
				Required:    true,
				Description: "The HTTP response header name (e.g. Access-Control-Allow-Origin, Cache-Control).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"value": schema.StringAttribute{
				Required:    true,
				Description: "The HTTP response header value.",
			},
		},
	}
}

func (r *LiveDomainHttpHeaderResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*ClientConfig)
}

func (r *LiveDomainHttpHeaderResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan LiveDomainHttpHeaderModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	live, err := newLiveClientFromConfig(r.client)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create Live client", err.Error())
		return
	}

	domainName := plan.DomainName.ValueString()
	if err := batchSetConfig(live, domainName, "set_resp_header", map[string]string{
		"key":   plan.Key.ValueString(),
		"value": plan.Value.ValueString(),
	}); err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to set HTTP header for domain %q", domainName), err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *LiveDomainHttpHeaderResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state LiveDomainHttpHeaderModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	live, err := newLiveClientFromConfig(r.client)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create Live client", err.Error())
		return
	}

	domainName := state.DomainName.ValueString()
	configs, err := describeAllFunctionConfigs(live, domainName, "set_resp_header")
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to read HTTP headers for domain %q", domainName), err.Error())
		return
	}

	// Find the header matching this resource's key.
	for _, cfg := range configs {
		if cfg["key"] == state.Key.ValueString() {
			state.Value = types.StringValue(cfg["value"])
			resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
			return
		}
	}

	// Header no longer exists.
	resp.State.RemoveResource(ctx)
}

func (r *LiveDomainHttpHeaderResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan LiveDomainHttpHeaderModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	live, err := newLiveClientFromConfig(r.client)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create Live client", err.Error())
		return
	}

	domainName := plan.DomainName.ValueString()
	if err := batchSetConfig(live, domainName, "set_resp_header", map[string]string{
		"key":   plan.Key.ValueString(),
		"value": plan.Value.ValueString(),
	}); err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to update HTTP header for domain %q", domainName), err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *LiveDomainHttpHeaderResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state LiveDomainHttpHeaderModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	live, err := newLiveClientFromConfig(r.client)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create Live client", err.Error())
		return
	}

	// Remove this specific header by re-setting all remaining headers (excluding this key).
	domainName := state.DomainName.ValueString()
	configs, err := describeAllFunctionConfigs(live, domainName, "set_resp_header")
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to read HTTP headers for domain %q", domainName), err.Error())
		return
	}

	// Delete all, then re-add the ones that should remain.
	if err := batchDeleteConfig(live, domainName, "set_resp_header"); err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to clear HTTP headers for domain %q", domainName), err.Error())
		return
	}

	for _, cfg := range configs {
		if cfg["key"] == state.Key.ValueString() {
			continue // skip the one being deleted
		}
		if err := batchSetConfig(live, domainName, "set_resp_header", map[string]string{
			"key":   cfg["key"],
			"value": cfg["value"],
		}); err != nil {
			resp.Diagnostics.AddError(fmt.Sprintf("Failed to restore HTTP header %q for domain %q", cfg["key"], domainName), err.Error())
			return
		}
	}
}

func (r *LiveDomainHttpHeaderResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import format: <domain_name>:<header_key>
	parts := splitImportID(req.ID, 2)
	if parts == nil {
		resp.Diagnostics.AddError("Invalid import ID", "Expected format: <domain_name>:<header_key>")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("domain_name"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("key"), parts[1])...)
}
