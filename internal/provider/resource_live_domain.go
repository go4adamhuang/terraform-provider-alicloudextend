package provider

import (
	"context"
	"fmt"
	"strings"

	openapiutil "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	liveclient "github.com/alibabacloud-go/live-20161101/v2/client"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &LiveDomainResource{}
var _ resource.ResourceWithImportState = &LiveDomainResource{}

type LiveDomainResource struct {
	client *ClientConfig
}

func NewLiveDomainResource() resource.Resource {
	return &LiveDomainResource{}
}

type LiveDomainModel struct {
	DomainName     types.String `tfsdk:"domain_name"`
	LiveDomainType types.String `tfsdk:"live_domain_type"`
	Region         types.String `tfsdk:"region"`
	Scope          types.String `tfsdk:"scope"`
	CheckUrl       types.String `tfsdk:"check_url"`
	TopLevelDomain types.String `tfsdk:"top_level_domain"`
	// Computed
	Cname        types.String `tfsdk:"cname"`
	DomainStatus types.String `tfsdk:"domain_status"`
	GmtCreated   types.String `tfsdk:"gmt_created"`
}

func (r *LiveDomainResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_live_domain"
}

func (r *LiveDomainResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	forceNew := []planmodifier.String{stringplanmodifier.RequiresReplace()}

	resp.Schema = schema.Schema{
		Description: "Manages an ApsaraVideo Live domain (ingest or streaming). " +
			"Before creating this resource, add the TXT record returned by the " +
			"`alicloudextend_live_domain_verify_content` data source to your DNS and ensure " +
			"domain ownership verification passes.",
		Attributes: map[string]schema.Attribute{
			"domain_name": schema.StringAttribute{
				Required:      true,
				Description:   "The live domain name (e.g. live.example.com).",
				PlanModifiers: forceNew,
			},
			"live_domain_type": schema.StringAttribute{
				Required:      true,
				Description:   "The type of live domain. Valid values: liveVideo (streaming domain), liveEdge (ingest domain).",
				PlanModifiers: forceNew,
			},
			"region": schema.StringAttribute{
				Required:      true,
				Description:   "The region where the domain resides (e.g. cn-shanghai, ap-southeast-1).",
				PlanModifiers: forceNew,
			},
			"scope": schema.StringAttribute{
				Optional:      true,
				Description:   "The acceleration region. Valid values: domestic, overseas, global. Defaults to domestic.",
				PlanModifiers: forceNew,
			},
			"check_url": schema.StringAttribute{
				Optional:    true,
				Description: "The URL used for health checks (e.g. http://live.example.com/status.html).",
			},
			"top_level_domain": schema.StringAttribute{
				Optional:      true,
				Description:   "The top-level domain name.",
				PlanModifiers: forceNew,
			},
			// Computed
			"cname": schema.StringAttribute{
				Computed:    true,
				Description: "The CNAME assigned to the domain. Point your domain's CNAME record to this value.",
			},
			"domain_status": schema.StringAttribute{
				Computed:    true,
				Description: "The domain status (online, offline, configuring).",
			},
			"gmt_created": schema.StringAttribute{
				Computed:    true,
				Description: "The time when the domain was created (ISO 8601 UTC).",
			},
		},
	}
}

func (r *LiveDomainResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*ClientConfig)
}

func (r *LiveDomainResource) newLiveClient() (*liveclient.Client, error) {
	regionID := "cn-hangzhou"
	if r.client.Region != "" {
		regionID = r.client.Region
	}
	return liveclient.NewClient(&openapiutil.Config{
		AccessKeyId:     strPtr(r.client.AccessKeyID),
		AccessKeySecret: strPtr(r.client.AccessKeySecret),
		Endpoint:        strPtr("live.aliyuncs.com"),
		RegionId:        strPtr(regionID),
	})
}

func (r *LiveDomainResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan LiveDomainModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	live, err := r.newLiveClient()
	if err != nil {
		resp.Diagnostics.AddError("Failed to create Live client", err.Error())
		return
	}

	domainName := plan.DomainName.ValueString()

	// Step 1: Verify domain ownership via DNS TXT record.
	verifyType := "dnsCheck"
	_, err = live.VerifyLiveDomainOwner(&liveclient.VerifyLiveDomainOwnerRequest{
		DomainName: strPtr(domainName),
		VerifyType: strPtr(verifyType),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Domain ownership verification failed for %q", domainName),
			"Ensure the DNS TXT record from alicloudextend_live_domain_verify_content has propagated, then retry.\n\nError: "+err.Error(),
		)
		return
	}

	// Step 2: Add the live domain.
	addReq := &liveclient.AddLiveDomainRequest{
		DomainName:     strPtr(domainName),
		LiveDomainType: strPtr(plan.LiveDomainType.ValueString()),
		Region:         strPtr(plan.Region.ValueString()),
	}
	if !plan.Scope.IsNull() && !plan.Scope.IsUnknown() {
		addReq.Scope = strPtr(plan.Scope.ValueString())
	}
	if !plan.CheckUrl.IsNull() && !plan.CheckUrl.IsUnknown() {
		addReq.CheckUrl = strPtr(plan.CheckUrl.ValueString())
	}
	if !plan.TopLevelDomain.IsNull() && !plan.TopLevelDomain.IsUnknown() {
		addReq.TopLevelDomain = strPtr(plan.TopLevelDomain.ValueString())
	}

	_, err = live.AddLiveDomain(addReq)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to add live domain %q", domainName), err.Error())
		return
	}

	// Step 3: Read back to populate computed fields.
	r.readIntoModel(ctx, live, domainName, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *LiveDomainResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state LiveDomainModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	live, err := r.newLiveClient()
	if err != nil {
		resp.Diagnostics.AddError("Failed to create Live client", err.Error())
		return
	}

	domainName := state.DomainName.ValueString()
	detail, err := live.DescribeLiveDomainDetail(&liveclient.DescribeLiveDomainDetailRequest{
		DomainName: strPtr(domainName),
	})
	if err != nil {
		if isLiveNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to describe live domain %q", domainName), err.Error())
		return
	}

	if detail.Body == nil || detail.Body.DomainDetail == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	d := detail.Body.DomainDetail
	if d.Cname != nil {
		state.Cname = types.StringValue(*d.Cname)
	}
	if d.DomainStatus != nil {
		state.DomainStatus = types.StringValue(*d.DomainStatus)
	}
	if d.GmtCreated != nil {
		state.GmtCreated = types.StringValue(*d.GmtCreated)
	}
	if d.LiveDomainType != nil {
		state.LiveDomainType = types.StringValue(*d.LiveDomainType)
	}
	if d.Region != nil {
		state.Region = types.StringValue(*d.Region)
	}
	if d.Scope != nil {
		state.Scope = types.StringValue(*d.Scope)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *LiveDomainResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// All mutable fields (check_url) do not have a dedicated update API in ApsaraVideo Live;
	// the domain must be deleted and recreated. ForceNew handles immutable fields.
	// For check_url changes, we simply reflect the new plan value in state since it is
	// only used at creation time.
	var plan LiveDomainModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var state LiveDomainModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Carry computed fields forward.
	plan.Cname = state.Cname
	plan.DomainStatus = state.DomainStatus
	plan.GmtCreated = state.GmtCreated

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *LiveDomainResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state LiveDomainModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	live, err := r.newLiveClient()
	if err != nil {
		resp.Diagnostics.AddError("Failed to create Live client", err.Error())
		return
	}

	domainName := state.DomainName.ValueString()
	_, err = live.DeleteLiveDomain(&liveclient.DeleteLiveDomainRequest{
		DomainName: strPtr(domainName),
	})
	if err != nil && !isLiveNotFound(err) {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to delete live domain %q", domainName), err.Error())
	}
}

func (r *LiveDomainResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("domain_name"), req, resp)
}

// readIntoModel populates computed fields from DescribeLiveDomainDetail into model.
func (r *LiveDomainResource) readIntoModel(ctx context.Context, live *liveclient.Client, domainName string, m *LiveDomainModel, diags interface {
	AddError(summary, detail string)
	HasError() bool
}) {
	detail, err := live.DescribeLiveDomainDetail(&liveclient.DescribeLiveDomainDetailRequest{
		DomainName: strPtr(domainName),
	})
	if err != nil {
		diags.AddError(fmt.Sprintf("Failed to describe live domain %q after creation", domainName), err.Error())
		return
	}
	if detail.Body == nil || detail.Body.DomainDetail == nil {
		return
	}
	d := detail.Body.DomainDetail
	if d.Cname != nil {
		m.Cname = types.StringValue(*d.Cname)
	}
	if d.DomainStatus != nil {
		m.DomainStatus = types.StringValue(*d.DomainStatus)
	}
	if d.GmtCreated != nil {
		m.GmtCreated = types.StringValue(*d.GmtCreated)
	}
}

// isLiveNotFound returns true when the error indicates the domain does not exist.
func isLiveNotFound(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "InvalidDomain.NotFound") ||
		strings.Contains(msg, "DomainNotFound") ||
		strings.Contains(msg, "404")
}
