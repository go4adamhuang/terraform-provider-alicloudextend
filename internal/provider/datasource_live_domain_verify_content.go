package provider

import (
	"context"
	"fmt"

	openapiutil "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	liveclient "github.com/alibabacloud-go/live-20161101/v2/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &LiveDomainVerifyContentDataSource{}

type LiveDomainVerifyContentDataSource struct {
	client *ClientConfig
}

func NewLiveDomainVerifyContentDataSource() datasource.DataSource {
	return &LiveDomainVerifyContentDataSource{}
}

type LiveDomainVerifyContentModel struct {
	DomainName types.String `tfsdk:"domain_name"`
	Content    types.String `tfsdk:"content"`
}

func (d *LiveDomainVerifyContentDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_live_domain_verify_content"
}

func (d *LiveDomainVerifyContentDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves the TXT record verification content for an ApsaraVideo Live domain. " +
			"Add this content as a DNS TXT record under `_dnsauth.<domain>` before creating the live domain resource.",
		Attributes: map[string]schema.Attribute{
			"domain_name": schema.StringAttribute{
				Required:    true,
				Description: "The domain name to query verification content for (e.g. live.example.com).",
			},
			"content": schema.StringAttribute{
				Computed:    true,
				Description: "The TXT record value to add to DNS (e.g. verify_dffeb6610035dcb77b413a59c32c****).",
			},
		},
	}
}

func (d *LiveDomainVerifyContentDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	d.client = req.ProviderData.(*ClientConfig)
}

func (d *LiveDomainVerifyContentDataSource) newLiveClient() (*liveclient.Client, error) {
	regionID := "cn-hangzhou"
	if d.client.Region != "" {
		regionID = d.client.Region
	}
	return liveclient.NewClient(&openapiutil.Config{
		AccessKeyId:     strPtr(d.client.AccessKeyID),
		AccessKeySecret: strPtr(d.client.AccessKeySecret),
		Endpoint:        strPtr("live.aliyuncs.com"),
		RegionId:        strPtr(regionID),
	})
}

func (d *LiveDomainVerifyContentDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state LiveDomainVerifyContentModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	live, err := d.newLiveClient()
	if err != nil {
		resp.Diagnostics.AddError("Failed to create Live client", err.Error())
		return
	}

	domainName := state.DomainName.ValueString()
	verifyResp, err := live.DescribeLiveVerifyContent(&liveclient.DescribeLiveVerifyContentRequest{
		DomainName: strPtr(domainName),
	})
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to get verify content for domain %q", domainName), err.Error())
		return
	}

	if verifyResp.Body == nil || verifyResp.Body.Content == nil {
		resp.Diagnostics.AddError("Empty response", "DescribeLiveVerifyContent returned no content")
		return
	}

	state.Content = types.StringValue(*verifyResp.Body.Content)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
