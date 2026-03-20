package provider

import (
	"context"
	"fmt"

	openapiutil "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	esaclient "github.com/alibabacloud-go/esa-20240910/v2/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &EsaRatePlanInstancesDataSource{}

type EsaRatePlanInstancesDataSource struct {
	client *ClientConfig
}

func NewEsaRatePlanInstancesDataSource() datasource.DataSource {
	return &EsaRatePlanInstancesDataSource{}
}

type EsaRatePlanInstancesModel struct {
	PlanNameEn              types.String               `tfsdk:"plan_name_en"`
	PlanType                types.String               `tfsdk:"plan_type"`
	Status                  types.String               `tfsdk:"status"`
	CheckRemainingSiteQuota types.Bool                 `tfsdk:"check_remaining_site_quota"`
	InstanceId              types.String               `tfsdk:"instance_id"`
	Instances               []EsaRatePlanInstanceModel `tfsdk:"instances"`
}

type EsaRatePlanInstanceModel struct {
	InstanceId types.String `tfsdk:"instance_id"`
	PlanName   types.String `tfsdk:"plan_name"`
	PlanType   types.String `tfsdk:"plan_type"`
	Status     types.String `tfsdk:"status"`
	SiteQuota  types.String `tfsdk:"site_quota"`
	ExpireTime types.String `tfsdk:"expire_time"`
	Coverages  types.String `tfsdk:"coverages"`
}

func (d *EsaRatePlanInstancesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_esa_rate_plan_instances"
}

func (d *EsaRatePlanInstancesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists ESA rate plan instances. Returns the first matching instance_id for easy use in other resources.",
		Attributes: map[string]schema.Attribute{
			"plan_name_en": schema.StringAttribute{
				Optional:    true,
				Description: "Filter by plan name (e.g. vipplan_intl). Defaults to \"vipplan_intl\".",
			},
			"plan_type": schema.StringAttribute{
				Optional:    true,
				Description: "Filter by plan type. Valid values: normal, enterprise. Leave unset to return all types.",
			},
			"status": schema.StringAttribute{
				Optional:    true,
				Description: "Filter by plan status: online, offline, disable, overdue. Defaults to \"online\".",
			},
			"check_remaining_site_quota": schema.BoolAttribute{
				Optional:    true,
				Description: "Only return instances with remaining site quota. Defaults to true.",
			},
			"instance_id": schema.StringAttribute{
				Computed:    true,
				Description: "The instance_id of the first matching plan instance.",
			},
			"instances": schema.ListNestedAttribute{
				Computed:    true,
				Description: "All matching plan instances.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"instance_id": schema.StringAttribute{
							Computed:    true,
							Description: "Plan instance ID.",
						},
						"plan_name": schema.StringAttribute{
							Computed:    true,
							Description: "Plan name.",
						},
						"plan_type": schema.StringAttribute{
							Computed:    true,
							Description: "Plan type.",
						},
						"status": schema.StringAttribute{
							Computed:    true,
							Description: "Plan status.",
						},
						"site_quota": schema.StringAttribute{
							Computed:    true,
							Description: "Maximum number of websites that can be associated with the plan.",
						},
						"expire_time": schema.StringAttribute{
							Computed:    true,
							Description: "Plan expiration time (RFC3339).",
						},
						"coverages": schema.StringAttribute{
							Computed:    true,
							Description: "Service locations covered by the plan (e.g. domestic,overseas,global).",
						},
					},
				},
			},
		},
	}
}

func (d *EsaRatePlanInstancesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	d.client = req.ProviderData.(*ClientConfig)
}

func (d *EsaRatePlanInstancesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state EsaRatePlanInstancesModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// ESA is a global service; use a fixed regional endpoint.
	endpoint := "esa.cn-hangzhou.aliyuncs.com"
	if d.client.Region != "" {
		endpoint = fmt.Sprintf("esa.%s.aliyuncs.com", d.client.Region)
	}

	cfg := &openapiutil.Config{
		AccessKeyId:     strPtr(d.client.AccessKeyID),
		AccessKeySecret: strPtr(d.client.AccessKeySecret),
		Endpoint:        strPtr(endpoint),
	}
	esa, err := esaclient.NewClient(cfg)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create ESA client", err.Error())
		return
	}

	// Resolve filter values with defaults.
	planNameEn := "vipplan_intl"
	if !state.PlanNameEn.IsNull() && state.PlanNameEn.ValueString() != "" {
		planNameEn = state.PlanNameEn.ValueString()
	}
	planType := ""
	if !state.PlanType.IsNull() && state.PlanType.ValueString() != "" {
		planType = state.PlanType.ValueString()
	}
	status := "online"
	if !state.Status.IsNull() && state.Status.ValueString() != "" {
		status = state.Status.ValueString()
	}
	checkQuota := true
	if !state.CheckRemainingSiteQuota.IsNull() {
		checkQuota = state.CheckRemainingSiteQuota.ValueBool()
	}
	checkQuotaStr := "false"
	if checkQuota {
		checkQuotaStr = "true"
	}

	// Paginate through all results.
	// PlanNameEn and PlanType are not supported as API-side filters (causes InvalidParameter);
	// fetch all matching status/quota results and filter client-side.
	var allInstances []*esaclient.ListUserRatePlanInstancesResponseBodyInstanceInfo
	pageNum := int32(1)
	pageSize := int32(100)
	for {
		listReq := &esaclient.ListUserRatePlanInstancesRequest{
			Status:                  strPtr(status),
			CheckRemainingSiteQuota: strPtr(checkQuotaStr),
			PageNumber:              &pageNum,
			PageSize:                &pageSize,
		}
		listResp, err := esa.ListUserRatePlanInstances(listReq)
		if err != nil {
			resp.Diagnostics.AddError("Failed to list ESA rate plan instances", err.Error())
			return
		}
		if listResp.Body == nil {
			break
		}
		allInstances = append(allInstances, listResp.Body.InstanceInfo...)

		totalPage := int32(0)
		if listResp.Body.TotalPage != nil {
			totalPage = *listResp.Body.TotalPage
		}
		if pageNum >= totalPage {
			break
		}
		pageNum++
	}

	// Map to state model, applying client-side plan_name_en / plan_type filters.
	instances := make([]EsaRatePlanInstanceModel, 0, len(allInstances))
	for _, inst := range allInstances {
		if planNameEn != "" && inst.PlanName != nil && *inst.PlanName != planNameEn {
			continue
		}
		if planType != "" && inst.PlanType != nil && *inst.PlanType != planType {
			continue
		}
		instances = append(instances, EsaRatePlanInstanceModel{
			InstanceId: types.StringPointerValue(inst.InstanceId),
			PlanName:   types.StringPointerValue(inst.PlanName),
			PlanType:   types.StringPointerValue(inst.PlanType),
			Status:     types.StringPointerValue(inst.Status),
			SiteQuota:  types.StringPointerValue(inst.SiteQuota),
			ExpireTime: types.StringPointerValue(inst.ExpireTime),
			Coverages:  types.StringPointerValue(inst.Coverages),
		})
	}
	state.Instances = instances

	if len(instances) > 0 {
		state.InstanceId = instances[0].InstanceId
	} else {
		state.InstanceId = types.StringValue("")
	}

	// Persist resolved defaults back to state for plan stability.
	state.PlanNameEn = types.StringValue(planNameEn)
	if planType != "" {
		state.PlanType = types.StringValue(planType)
	}
	state.Status = types.StringValue(status)
	state.CheckRemainingSiteQuota = types.BoolValue(checkQuota)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
