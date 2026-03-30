package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ provider.Provider = &AliCloudProvider{}
var _ provider.ProviderWithFunctions = &AliCloudProvider{}

type AliCloudProvider struct {
	version string
}

type AliCloudProviderModel struct {
	AccessKeyID     types.String `tfsdk:"access_key_id"`
	AccessKeySecret types.String `tfsdk:"access_key_secret"`
	Region          types.String `tfsdk:"region"`
	Profile         types.String `tfsdk:"profile"`
}

// ClientConfig holds resolved credentials passed to every resource and data source.
type ClientConfig struct {
	AccessKeyID     string
	AccessKeySecret string
	Region          string
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &AliCloudProvider{version: version}
	}
}

func (p *AliCloudProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "alicloudextend"
	resp.Version = p.version
}

func (p *AliCloudProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Extended Alibaba Cloud provider (alicloudextend) for resources and data sources not covered by, or rewritten from, the official aliyun/alicloud provider.",
		Attributes: map[string]schema.Attribute{
			"access_key_id": schema.StringAttribute{
				Optional:    true,
				Description: "Alibaba Cloud access key ID. Priority: explicit > profile > ALICLOUD_ACCESS_KEY_ID env var.",
			},
			"access_key_secret": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "Alibaba Cloud access key secret. Priority: explicit > profile > ALICLOUD_ACCESS_KEY_SECRET env var.",
			},
			"region": schema.StringAttribute{
				Optional:    true,
				Description: "Alibaba Cloud region ID (e.g. cn-hangzhou). Priority: explicit > profile > ALICLOUD_REGION_ID env var.",
			},
			"profile": schema.StringAttribute{
				Optional:    true,
				Description: "Alibaba Cloud CLI profile name. Loads credentials from ~/.aliyun/config.json. Falls back to ALICLOUD_PROFILE env var.",
			},
		},
	}
}

func (p *AliCloudProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config AliCloudProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	accessKeyID := os.Getenv("ALICLOUD_ACCESS_KEY_ID")
	accessKeySecret := os.Getenv("ALICLOUD_ACCESS_KEY_SECRET")
	region := os.Getenv("ALICLOUD_REGION_ID")

	profileName := os.Getenv("ALICLOUD_PROFILE")
	if !config.Profile.IsNull() && config.Profile.ValueString() != "" {
		profileName = config.Profile.ValueString()
	}
	if profileName != "" {
		pID, pSecret, pRegion, err := loadAliyunProfile(profileName)
		if err != nil {
			resp.Diagnostics.AddError(fmt.Sprintf("Failed to load Alibaba Cloud CLI profile %q", profileName), err.Error())
			return
		}
		if pID != "" {
			accessKeyID = pID
		}
		if pSecret != "" {
			accessKeySecret = pSecret
		}
		if pRegion != "" {
			region = pRegion
		}
	}
	if !config.AccessKeyID.IsNull() && config.AccessKeyID.ValueString() != "" {
		accessKeyID = config.AccessKeyID.ValueString()
	}
	if !config.AccessKeySecret.IsNull() && config.AccessKeySecret.ValueString() != "" {
		accessKeySecret = config.AccessKeySecret.ValueString()
	}
	if !config.Region.IsNull() && config.Region.ValueString() != "" {
		region = config.Region.ValueString()
	}

	clientCfg := &ClientConfig{
		AccessKeyID:     accessKeyID,
		AccessKeySecret: accessKeySecret,
		Region:          region,
	}

	resp.DataSourceData = clientCfg
	resp.ResourceData = clientCfg
}

func (p *AliCloudProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewEsaHttpsBasicConfigurationResource,
		NewLiveDomainResource,
		NewLiveDomainCertificateResource,
		NewLiveDomainHttpsOptionResource,
		NewLiveDomainHttpHeaderResource,
		NewLiveDomainAccessControlResource,
		NewLiveDomainIpv6Resource,
	}
}

func (p *AliCloudProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewEsaRatePlanInstancesDataSource,
		NewEsaSitesDataSource,
		NewLiveDomainVerifyContentDataSource,
	}
}

func (p *AliCloudProvider) Functions(_ context.Context) []func() function.Function {
	return []func() function.Function{}
}

// loadAliyunProfile reads credentials from the Alibaba Cloud CLI config file (~/.aliyun/config.json).
func loadAliyunProfile(profileName string) (accessKeyID, accessKeySecret, region string, err error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", "", "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	cfgPath := filepath.Join(home, ".aliyun", "config.json")
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		return "", "", "", fmt.Errorf("cannot read %s: %w", cfgPath, err)
	}

	var cfg struct {
		Profiles []struct {
			Name            string `json:"name"`
			AccessKeyID     string `json:"access_key_id"`
			AccessKeySecret string `json:"access_key_secret"`
			RegionID        string `json:"region_id"`
		} `json:"profiles"`
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return "", "", "", fmt.Errorf("cannot parse %s: %w", cfgPath, err)
	}

	for _, p := range cfg.Profiles {
		if p.Name == profileName {
			return p.AccessKeyID, p.AccessKeySecret, p.RegionID, nil
		}
	}
	return "", "", "", fmt.Errorf("profile %q not found in %s", profileName, cfgPath)
}
