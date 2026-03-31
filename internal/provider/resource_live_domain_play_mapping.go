package provider

import (
	"context"
	"fmt"
	"strings"

	liveclient "github.com/alibabacloud-go/live-20161101/v2/client"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &LiveDomainPlayMappingResource{}
var _ resource.ResourceWithImportState = &LiveDomainPlayMappingResource{}

type LiveDomainPlayMappingResource struct{ client *ClientConfig }

func NewLiveDomainPlayMappingResource() resource.Resource {
	return &LiveDomainPlayMappingResource{}
}

type LiveDomainPlayMappingModel struct {
	SubDomain  types.String `tfsdk:"sub_domain"`
	RootDomain types.String `tfsdk:"root_domain"`
}

func (r *LiveDomainPlayMappingResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_live_domain_play_mapping"
}

func (r *LiveDomainPlayMappingResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Binds a sub-streaming domain to a main streaming domain for ApsaraVideo Live.",
		Attributes: map[string]schema.Attribute{
			"sub_domain": schema.StringAttribute{
				Required:    true,
				Description: "The sub-streaming domain (liveVideo type) to bind.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"root_domain": schema.StringAttribute{
				Required:    true,
				Description: "The main streaming domain to bind to.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *LiveDomainPlayMappingResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*ClientConfig)
}

func (r *LiveDomainPlayMappingResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan LiveDomainPlayMappingModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	live, err := newLiveClientFromConfig(r.client)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create Live client", err.Error())
		return
	}

	_, err = live.AddLiveDomainPlayMapping(&liveclient.AddLiveDomainPlayMappingRequest{
		PlayDomain: strPtr(plan.SubDomain.ValueString()),
		PullDomain: strPtr(plan.RootDomain.ValueString()),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Failed to bind sub domain %q to root domain %q", plan.SubDomain.ValueString(), plan.RootDomain.ValueString()),
			err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *LiveDomainPlayMappingResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state LiveDomainPlayMappingModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	live, err := newLiveClientFromConfig(r.client)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create Live client", err.Error())
		return
	}

	exists, err := playMappingExists(live, state.RootDomain.ValueString(), state.SubDomain.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Failed to read play mapping for domain %q", state.SubDomain.ValueString()),
			err.Error(),
		)
		return
	}
	if !exists {
		// Mapping no longer exists — remove from state.
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *LiveDomainPlayMappingResource) Update(_ context.Context, _ resource.UpdateRequest, _ *resource.UpdateResponse) {
	// All attributes have RequiresReplace; Update is never called.
}

func (r *LiveDomainPlayMappingResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state LiveDomainPlayMappingModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	live, err := newLiveClientFromConfig(r.client)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create Live client", err.Error())
		return
	}

	_, err = live.DeleteLiveDomainPlayMapping(&liveclient.DeleteLiveDomainPlayMappingRequest{
		PlayDomain: strPtr(state.SubDomain.ValueString()),
		PullDomain: strPtr(state.RootDomain.ValueString()),
	})
	if err != nil && !strings.Contains(err.Error(), "InvalidDomain.NotFound") {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Failed to unbind sub domain %q from root domain %q", state.SubDomain.ValueString(), state.RootDomain.ValueString()),
			err.Error(),
		)
	}
}

func (r *LiveDomainPlayMappingResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import format: <play_domain>:<pull_domain>
	parts := strings.SplitN(req.ID, ":", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			"Expected format: <play_domain>:<pull_domain>",
		)
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("sub_domain"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("root_domain"), parts[1])...)
}

// playMappingExists queries DescribeLiveDomainMapping using the pull domain (root_domain)
// and checks whether the given play domain (sub_domain) is listed as a mapped play domain.
func playMappingExists(live *liveclient.Client, pullDomain, playDomain string) (bool, error) {
	resp, err := live.DescribeLiveDomainMapping(&liveclient.DescribeLiveDomainMappingRequest{
		DomainName: strPtr(pullDomain),
	})
	if err != nil {
		if strings.Contains(err.Error(), "InvalidDomain.NotFound") {
			return false, nil
		}
		return false, err
	}
	if resp.Body == nil || resp.Body.LiveDomainModels == nil {
		return false, nil
	}
	for _, m := range resp.Body.LiveDomainModels.LiveDomainModel {
		if m.DomainName != nil && *m.DomainName == playDomain {
			return true, nil
		}
	}
	return false, nil
}
