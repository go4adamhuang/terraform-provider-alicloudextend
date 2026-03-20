package provider

import (
	"context"
	"fmt"
	"strconv"

	openapiutil "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	esaclient "github.com/alibabacloud-go/esa-20240910/v2/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &EsaSitesDataSource{}

type EsaSitesDataSource struct {
	client *ClientConfig
}

func NewEsaSitesDataSource() datasource.DataSource {
	return &EsaSitesDataSource{}
}

type EsaSitesModel struct {
	Sites []EsaSiteModel `tfsdk:"sites"`
}

type EsaSiteModel struct {
	SiteId     types.String `tfsdk:"site_id"`
	SiteName   types.String `tfsdk:"site_name"`
	InstanceId types.String `tfsdk:"instance_id"`
	Status     types.String `tfsdk:"status"`
}

func (d *EsaSitesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_esa_sites"
}

func (d *EsaSitesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists all ESA sites under the account.",
		Attributes: map[string]schema.Attribute{
			"sites": schema.ListNestedAttribute{
				Computed:    true,
				Description: "All ESA sites.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"site_id": schema.StringAttribute{
							Computed:    true,
							Description: "The numeric site ID.",
						},
						"site_name": schema.StringAttribute{
							Computed:    true,
							Description: "The domain name (e.g. example.com).",
						},
						"instance_id": schema.StringAttribute{
							Computed:    true,
							Description: "The bound rate plan instance ID.",
						},
						"status": schema.StringAttribute{
							Computed:    true,
							Description: "The site status (e.g. active, pending, offline, moved).",
						},
					},
				},
			},
		},
	}
}

func (d *EsaSitesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	d.client = req.ProviderData.(*ClientConfig)
}

func (d *EsaSitesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state EsaSitesModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	endpoint := "esa.cn-hangzhou.aliyuncs.com"
	if d.client.Region != "" {
		endpoint = fmt.Sprintf("esa.%s.aliyuncs.com", d.client.Region)
	}
	esa, err := esaclient.NewClient(&openapiutil.Config{
		AccessKeyId:     strPtr(d.client.AccessKeyID),
		AccessKeySecret: strPtr(d.client.AccessKeySecret),
		Endpoint:        strPtr(endpoint),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create ESA client", err.Error())
		return
	}

	pageSize := int32(500)
	var allSites []*esaclient.ListSitesResponseBodySites
	pageNum := int32(1)

	for {
		listResp, err := esa.ListSites(&esaclient.ListSitesRequest{
			PageNumber: &pageNum,
			PageSize:   &pageSize,
		})
		if err != nil {
			resp.Diagnostics.AddError("Failed to list ESA sites", err.Error())
			return
		}
		if listResp.Body == nil {
			break
		}
		allSites = append(allSites, listResp.Body.Sites...)

		totalCount := int32(0)
		if listResp.Body.TotalCount != nil {
			totalCount = *listResp.Body.TotalCount
		}
		fetched := pageNum * pageSize
		if fetched >= totalCount {
			break
		}
		pageNum++
	}

	sites := make([]EsaSiteModel, 0, len(allSites))
	for _, s := range allSites {
		m := EsaSiteModel{
			SiteName:   types.StringPointerValue(s.SiteName),
			InstanceId: types.StringPointerValue(s.InstanceId),
			Status:     types.StringPointerValue(s.Status),
		}
		if s.SiteId != nil {
			m.SiteId = types.StringValue(strconv.FormatInt(*s.SiteId, 10))
		}
		sites = append(sites, m)
	}
	state.Sites = sites
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
