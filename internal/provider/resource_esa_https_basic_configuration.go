package provider

import (
	"context"
	"fmt"
	"strconv"

	openapiutil "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	esaclient "github.com/alibabacloud-go/esa-20240910/v2/client"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &EsaHttpsBasicConfigurationResource{}

type EsaHttpsBasicConfigurationResource struct {
	client *ClientConfig
}

func NewEsaHttpsBasicConfigurationResource() resource.Resource {
	return &EsaHttpsBasicConfigurationResource{}
}

type EsaHttpsBasicConfigurationModel struct {
	SiteId   types.String `tfsdk:"site_id"`
	Http2    types.String `tfsdk:"http2"`
	ConfigId types.String `tfsdk:"config_id"`
}

func (r *EsaHttpsBasicConfigurationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_esa_https_basic_configuration"
}

func (r *EsaHttpsBasicConfigurationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages ESA HTTPS basic configuration for a site. Uses upsert semantics on Create to handle the singleton configuration that is auto-created with each site, avoiding ConfExceedLimit errors.",
		Attributes: map[string]schema.Attribute{
			"site_id": schema.StringAttribute{
				Required:    true,
				Description: "The ESA site ID.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"http2": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether to enable HTTP2. Valid values: on, off. Defaults to off.",
			},
			"config_id": schema.StringAttribute{
				Computed:    true,
				Description: "The configuration ID assigned by ESA.",
			},
		},
	}
}

func (r *EsaHttpsBasicConfigurationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*ClientConfig)
}

func (r *EsaHttpsBasicConfigurationResource) newESAClient() (*esaclient.Client, error) {
	endpoint := "esa.cn-hangzhou.aliyuncs.com"
	if r.client.Region != "" {
		endpoint = fmt.Sprintf("esa.%s.aliyuncs.com", r.client.Region)
	}
	return esaclient.NewClient(&openapiutil.Config{
		AccessKeyId:     strPtr(r.client.AccessKeyID),
		AccessKeySecret: strPtr(r.client.AccessKeySecret),
		Endpoint:        strPtr(endpoint),
	})
}

func (r *EsaHttpsBasicConfigurationResource) listGlobalConfig(esa *esaclient.Client, siteId int64) ([]*esaclient.ListHttpsBasicConfigurationsResponseBodyConfigs, error) {
	configType := "global"
	listResp, err := esa.ListHttpsBasicConfigurations(&esaclient.ListHttpsBasicConfigurationsRequest{
		SiteId:     &siteId,
		ConfigType: &configType,
	})
	if err != nil {
		return nil, err
	}
	if listResp.Body == nil {
		return nil, nil
	}
	return listResp.Body.Configs, nil
}

func (r *EsaHttpsBasicConfigurationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan EsaHttpsBasicConfigurationModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	siteId, err := strconv.ParseInt(plan.SiteId.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid site_id", err.Error())
		return
	}

	http2 := "off"
	if !plan.Http2.IsNull() && !plan.Http2.IsUnknown() && plan.Http2.ValueString() != "" {
		http2 = plan.Http2.ValueString()
	}

	esa, err := r.newESAClient()
	if err != nil {
		resp.Diagnostics.AddError("Failed to create ESA client", err.Error())
		return
	}

	configs, err := r.listGlobalConfig(esa, siteId)
	if err != nil {
		resp.Diagnostics.AddError("Failed to list HTTPS basic configurations", err.Error())
		return
	}

	var configId int64
	if len(configs) > 0 {
		// Upsert: singleton already exists, update it.
		configId = *configs[0].ConfigId
		_, err = esa.UpdateHttpsBasicConfiguration(&esaclient.UpdateHttpsBasicConfigurationRequest{
			SiteId:   &siteId,
			ConfigId: &configId,
			Http2:    strPtr(http2),
		})
		if err != nil {
			resp.Diagnostics.AddError("Failed to update HTTPS basic configuration", err.Error())
			return
		}
	} else {
		createResp, err := esa.CreateHttpsBasicConfiguration(&esaclient.CreateHttpsBasicConfigurationRequest{
			SiteId: &siteId,
			Http2:  strPtr(http2),
		})
		if err != nil {
			resp.Diagnostics.AddError("Failed to create HTTPS basic configuration", err.Error())
			return
		}
		configId = *createResp.Body.ConfigId
	}

	plan.ConfigId = types.StringValue(strconv.FormatInt(configId, 10))
	plan.Http2 = types.StringValue(http2)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *EsaHttpsBasicConfigurationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state EsaHttpsBasicConfigurationModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	siteId, err := strconv.ParseInt(state.SiteId.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid site_id", err.Error())
		return
	}

	esa, err := r.newESAClient()
	if err != nil {
		resp.Diagnostics.AddError("Failed to create ESA client", err.Error())
		return
	}

	configs, err := r.listGlobalConfig(esa, siteId)
	if err != nil {
		resp.Diagnostics.AddError("Failed to list HTTPS basic configurations", err.Error())
		return
	}

	if len(configs) == 0 {
		resp.State.RemoveResource(ctx)
		return
	}

	cfg := configs[0]
	state.ConfigId = types.StringValue(strconv.FormatInt(*cfg.ConfigId, 10))
	if cfg.Http2 != nil {
		state.Http2 = types.StringValue(*cfg.Http2)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *EsaHttpsBasicConfigurationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan EsaHttpsBasicConfigurationModel
	var state EsaHttpsBasicConfigurationModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	siteId, err := strconv.ParseInt(state.SiteId.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid site_id", err.Error())
		return
	}
	configId, err := strconv.ParseInt(state.ConfigId.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid config_id", err.Error())
		return
	}

	esa, err := r.newESAClient()
	if err != nil {
		resp.Diagnostics.AddError("Failed to create ESA client", err.Error())
		return
	}

	http2 := plan.Http2.ValueString()
	_, err = esa.UpdateHttpsBasicConfiguration(&esaclient.UpdateHttpsBasicConfigurationRequest{
		SiteId:   &siteId,
		ConfigId: &configId,
		Http2:    strPtr(http2),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to update HTTPS basic configuration", err.Error())
		return
	}

	state.Http2 = types.StringValue(http2)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *EsaHttpsBasicConfigurationResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
	// no-op: ESA singleton configuration cannot be deleted; it lives with the site.
}
