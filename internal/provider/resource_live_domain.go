package provider

import (
	"context"
	"fmt"
	"strings"
	"time"

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
	// Required / ForceNew
	DomainName  types.String `tfsdk:"domain_name"`
	DomainType  types.String `tfsdk:"domain_type"`
	Region      types.String `tfsdk:"region"`
	// Optional ForceNew
	CheckUrl        types.String `tfsdk:"check_url"`
	ResourceGroupId types.String `tfsdk:"resource_group_id"`
	Scope           types.String `tfsdk:"scope"`
	// Optional mutable
	Status types.String            `tfsdk:"status"`
	Tags   map[string]types.String `tfsdk:"tags"`
	// Computed
	Cname      types.String `tfsdk:"cname"`
	CreateTime types.String `tfsdk:"create_time"`
}

func (r *LiveDomainResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_live_domain"
}

func (r *LiveDomainResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	forceNew := []planmodifier.String{stringplanmodifier.RequiresReplace()}

	resp.Schema = schema.Schema{
		Description: "Manages an ApsaraVideo Live domain (ingest or streaming). Aligns with the " +
			"official alicloud_live_domain resource and extends it by waiting for the domain to " +
			"reach 'online' status so that the CNAME value is available for use in downstream resources.\n\n" +
			"Before creating, add the DNS TXT record from `alicloudextend_live_domain_verify_content` " +
			"to prove domain ownership.",
		Attributes: map[string]schema.Attribute{
			"domain_name": schema.StringAttribute{
				Required:      true,
				Description:   "The live domain name (e.g. live.example.com).",
				PlanModifiers: forceNew,
			},
			"domain_type": schema.StringAttribute{
				Required:      true,
				Description:   "The domain business type. Valid values: liveVideo (streaming domain), liveEdge (ingest domain).",
				PlanModifiers: forceNew,
			},
			"region": schema.StringAttribute{
				Required:      true,
				Description:   "The region to which the domain belongs (e.g. cn-shanghai, ap-southeast-1).",
				PlanModifiers: forceNew,
			},
			"check_url": schema.StringAttribute{
				Optional:      true,
				Description:   "The URL used for health checks. Immutable after creation.",
				PlanModifiers: forceNew,
			},
			"resource_group_id": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The resource group ID.",
			},
			"scope": schema.StringAttribute{
				Optional:      true,
				Description:   "The acceleration region. Valid values: domestic, overseas, global.",
				PlanModifiers: forceNew,
			},
			"status": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The domain operational state. Valid values: online, offline.",
			},
			"tags": schema.MapAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Key-value pairs for resource labeling.",
			},
			// Extension: official provider does not return this
			"cname": schema.StringAttribute{
				Computed:    true,
				Description: "The CNAME assigned by AliCloud. Point your domain's DNS CNAME record to this value. " +
					"Available once the domain reaches 'online' status.",
			},
			"create_time": schema.StringAttribute{
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

	// Verify domain ownership via DNS TXT record.
	_, err = live.VerifyLiveDomainOwner(&liveclient.VerifyLiveDomainOwnerRequest{
		DomainName: strPtr(domainName),
		VerifyType: strPtr("dnsCheck"),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Domain ownership verification failed for %q", domainName),
			"Ensure the DNS TXT record from alicloudextend_live_domain_verify_content has propagated, then retry.\n\nError: "+err.Error(),
		)
		return
	}

	// Add the live domain.
	addReq := &liveclient.AddLiveDomainRequest{
		DomainName:     strPtr(domainName),
		LiveDomainType: strPtr(plan.DomainType.ValueString()),
		Region:         strPtr(plan.Region.ValueString()),
	}
	if !plan.Scope.IsNull() && !plan.Scope.IsUnknown() {
		addReq.Scope = strPtr(plan.Scope.ValueString())
	}
	if !plan.CheckUrl.IsNull() && !plan.CheckUrl.IsUnknown() {
		addReq.CheckUrl = strPtr(plan.CheckUrl.ValueString())
	}
	if !plan.ResourceGroupId.IsNull() && !plan.ResourceGroupId.IsUnknown() {
		addReq.ResourceGroupId = strPtr(plan.ResourceGroupId.ValueString())
	}
	if len(plan.Tags) > 0 {
		tags := make([]*liveclient.AddLiveDomainRequestTag, 0, len(plan.Tags))
		for k, v := range plan.Tags {
			k, v := k, v
			tags = append(tags, &liveclient.AddLiveDomainRequestTag{
				Key:   strPtr(k),
				Value: strPtr(v.ValueString()),
			})
		}
		addReq.Tag = tags
	}

	_, err = live.AddLiveDomain(addReq)
	if err != nil && !isLiveDomainAlreadyExist(err) {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to add live domain %q", domainName), err.Error())
		return
	}

	// Poll until online and CNAME is available (our extension).
	r.waitForOnlineAndPopulate(ctx, live, domainName, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// Handle explicit offline request after creation.
	if !plan.Status.IsNull() && !plan.Status.IsUnknown() && plan.Status.ValueString() == "offline" {
		if _, err := live.StopLiveDomain(&liveclient.StopLiveDomainRequest{DomainName: strPtr(domainName)}); err != nil {
			resp.Diagnostics.AddError(fmt.Sprintf("Failed to stop live domain %q", domainName), err.Error())
			return
		}
		plan.Status = types.StringValue("offline")
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
		state.Status = types.StringValue(*d.DomainStatus)
	}
	if d.GmtCreated != nil {
		state.CreateTime = types.StringValue(*d.GmtCreated)
	}
	if d.LiveDomainType != nil {
		state.DomainType = types.StringValue(*d.LiveDomainType)
	}
	if d.Region != nil {
		state.Region = types.StringValue(*d.Region)
	}
	if d.Scope != nil {
		state.Scope = types.StringValue(*d.Scope)
	}
	if d.ResourceGroupId != nil {
		state.ResourceGroupId = types.StringValue(*d.ResourceGroupId)
	}

	// Read tags.
	tagResp, err := live.ListLiveTagResources(&liveclient.ListLiveTagResourcesRequest{
		ResourceType: strPtr("DOMAIN"),
		ResourceId:   []*string{strPtr(domainName)},
	})
	if err == nil && tagResp.Body != nil && tagResp.Body.TagResources != nil {
		tags := make(map[string]types.String)
		for _, tr := range tagResp.Body.TagResources.TagResource {
			if tr.TagKey != nil && tr.TagValue != nil {
				tags[*tr.TagKey] = types.StringValue(*tr.TagValue)
			}
		}
		if len(tags) > 0 {
			state.Tags = tags
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *LiveDomainResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state LiveDomainModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
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

	// Update status if changed.
	planStatus := plan.Status.ValueString()
	stateStatus := state.Status.ValueString()
	if planStatus != stateStatus && planStatus != "" {
		switch planStatus {
		case "online":
			if _, err := live.StartLiveDomain(&liveclient.StartLiveDomainRequest{DomainName: strPtr(domainName)}); err != nil {
				resp.Diagnostics.AddError(fmt.Sprintf("Failed to start live domain %q", domainName), err.Error())
				return
			}
		case "offline":
			if _, err := live.StopLiveDomain(&liveclient.StopLiveDomainRequest{DomainName: strPtr(domainName)}); err != nil {
				resp.Diagnostics.AddError(fmt.Sprintf("Failed to stop live domain %q", domainName), err.Error())
				return
			}
		}
	}

	// Update tags: remove stale keys, add/update new ones.
	if err := r.reconcileTags(live, domainName, state.Tags, plan.Tags); err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to update tags for live domain %q", domainName), err.Error())
		return
	}

	// Carry computed fields forward.
	plan.Cname = state.Cname
	plan.CreateTime = state.CreateTime

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
		return
	}

	// Wait until the domain is fully removed to prevent DomainAlreadyExist on rapid recreate.
	r.waitForDeleted(ctx, live, domainName, &resp.Diagnostics)
}

func (r *LiveDomainResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("domain_name"), req, resp)
}

// waitForOnlineAndPopulate polls until domain_status == "online" and CNAME is set (max 10 min).
func (r *LiveDomainResource) waitForOnlineAndPopulate(ctx context.Context, live *liveclient.Client, domainName string, m *LiveDomainModel, diags interface {
	AddError(summary, detail string)
	HasError() bool
}) {
	const timeout = 10 * time.Minute
	const interval = 10 * time.Second
	deadline := time.Now().Add(timeout)

	for {
		detail, err := live.DescribeLiveDomainDetail(&liveclient.DescribeLiveDomainDetailRequest{
			DomainName: strPtr(domainName),
		})
		if err != nil {
			diags.AddError(fmt.Sprintf("Failed to describe live domain %q", domainName), err.Error())
			return
		}
		if detail.Body != nil && detail.Body.DomainDetail != nil {
			d := detail.Body.DomainDetail
			status := ""
			if d.DomainStatus != nil {
				status = *d.DomainStatus
			}
			if status == "online" && d.Cname != nil && *d.Cname != "" {
				m.Cname = types.StringValue(*d.Cname)
				m.Status = types.StringValue(status)
				if d.GmtCreated != nil {
					m.CreateTime = types.StringValue(*d.GmtCreated)
				}
				return
			}
		}

		if time.Now().After(deadline) {
			diags.AddError(
				fmt.Sprintf("Timed out waiting for live domain %q to become online", domainName),
				"Domain did not reach 'online' status within 10 minutes. "+
					"Run `terraform refresh` once the domain is active.",
			)
			return
		}

		select {
		case <-ctx.Done():
			diags.AddError("Context cancelled", ctx.Err().Error())
			return
		case <-time.After(interval):
		}
	}
}

// reconcileTags removes tags no longer in plan and adds/updates tags in plan.
func (r *LiveDomainResource) reconcileTags(live *liveclient.Client, domainName string, oldTags, newTags map[string]types.String) error {
	resourceType := "DOMAIN"

	// Keys to remove.
	var removeKeys []*string
	for k := range oldTags {
		if _, ok := newTags[k]; !ok {
			k := k
			removeKeys = append(removeKeys, strPtr(k))
		}
	}
	if len(removeKeys) > 0 {
		if _, err := live.UnTagLiveResources(&liveclient.UnTagLiveResourcesRequest{
			ResourceType: strPtr(resourceType),
			ResourceId:   []*string{strPtr(domainName)},
			TagKey:       removeKeys,
		}); err != nil {
			return err
		}
	}

	// Tags to add/update.
	var addTags []*liveclient.TagLiveResourcesRequestTag
	for k, v := range newTags {
		k, v := k, v
		addTags = append(addTags, &liveclient.TagLiveResourcesRequestTag{
			Key:   strPtr(k),
			Value: strPtr(v.ValueString()),
		})
	}
	if len(addTags) > 0 {
		if _, err := live.TagLiveResources(&liveclient.TagLiveResourcesRequest{
			ResourceType: strPtr(resourceType),
			ResourceId:   []*string{strPtr(domainName)},
			Tag:          addTags,
		}); err != nil {
			return err
		}
	}
	return nil
}

// waitForDeleted polls until DescribeLiveDomainDetail returns NotFound (max 5 min).
func (r *LiveDomainResource) waitForDeleted(ctx context.Context, live *liveclient.Client, domainName string, diags interface {
	AddError(summary, detail string)
}) {
	const timeout = 5 * time.Minute
	const interval = 5 * time.Second
	deadline := time.Now().Add(timeout)

	for {
		_, err := live.DescribeLiveDomainDetail(&liveclient.DescribeLiveDomainDetailRequest{
			DomainName: strPtr(domainName),
		})
		if err != nil && isLiveNotFound(err) {
			return
		}

		if time.Now().After(deadline) {
			diags.AddError(
				fmt.Sprintf("Timed out waiting for live domain %q to be deleted", domainName),
				"Domain did not disappear within 5 minutes.",
			)
			return
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(interval):
		}
	}
}

// isLiveNotFound returns true when the error indicates the domain does not exist.
func isLiveNotFound(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "InvalidDomain.NotFound") ||
		strings.Contains(msg, "DomainNotFound") ||
		strings.Contains(msg, "404")
}

// isLiveDomainAlreadyExist returns true when the domain already exists.
func isLiveDomainAlreadyExist(err error) bool {
	return strings.Contains(err.Error(), "DomainAlreadyExist")
}
