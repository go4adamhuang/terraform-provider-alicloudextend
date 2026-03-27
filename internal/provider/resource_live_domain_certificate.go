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

var _ resource.Resource = &LiveDomainCertificateResource{}
var _ resource.ResourceWithImportState = &LiveDomainCertificateResource{}

type LiveDomainCertificateResource struct{ client *ClientConfig }

func NewLiveDomainCertificateResource() resource.Resource {
	return &LiveDomainCertificateResource{}
}

type LiveDomainCertificateModel struct {
	DomainName  types.String `tfsdk:"domain_name"`
	SSLProtocol types.String `tfsdk:"ssl_protocol"`
	SSLPub      types.String `tfsdk:"ssl_pub"`
	SSLPri      types.String `tfsdk:"ssl_pri"`
	CertName    types.String `tfsdk:"cert_name"`
	CertType    types.String `tfsdk:"cert_type"`
	Http2       types.String `tfsdk:"http2"`
	// Computed
	CertDomainName types.String `tfsdk:"cert_domain_name"`
	CertExpireTime types.String `tfsdk:"cert_expire_time"`
}

func (r *LiveDomainCertificateResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_live_domain_certificate"
}

func (r *LiveDomainCertificateResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages the HTTPS certificate and HTTP/2 configuration for an ApsaraVideo Live domain.",
		Attributes: map[string]schema.Attribute{
			"domain_name": schema.StringAttribute{
				Required:    true,
				Description: "The live domain name.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"ssl_protocol": schema.StringAttribute{
				Required:    true,
				Description: "Whether to enable HTTPS. Valid values: on, off.",
			},
			"ssl_pub": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "The public key of the certificate (PEM format). Required when cert_type is upload.",
			},
			"ssl_pri": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "The private key of the certificate (PEM format). Required when cert_type is upload.",
			},
			"cert_name": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The certificate name.",
			},
			"cert_type": schema.StringAttribute{
				Optional:    true,
				Description: "The certificate type. Valid values: upload (custom certificate), cas (Certificate Management Service), free (free certificate).",
			},
			"http2": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether to enable HTTP/2. Valid values: on, off. Only effective when ssl_protocol is on.",
			},
			// Computed
			"cert_domain_name": schema.StringAttribute{
				Computed:    true,
				Description: "The domain name that the certificate is bound to.",
			},
			"cert_expire_time": schema.StringAttribute{
				Computed:    true,
				Description: "The expiration time of the certificate.",
			},
		},
	}
}

func (r *LiveDomainCertificateResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*ClientConfig)
}

func (r *LiveDomainCertificateResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan LiveDomainCertificateModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	live, err := newLiveClientFromConfig(r.client)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create Live client", err.Error())
		return
	}

	if err := r.applyCertificate(live, &plan); err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to set certificate for domain %q", plan.DomainName.ValueString()), err.Error())
		return
	}

	r.readIntoModel(live, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *LiveDomainCertificateResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state LiveDomainCertificateModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	live, err := newLiveClientFromConfig(r.client)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create Live client", err.Error())
		return
	}

	r.readIntoModel(live, &state, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *LiveDomainCertificateResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan LiveDomainCertificateModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	live, err := newLiveClientFromConfig(r.client)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create Live client", err.Error())
		return
	}

	if err := r.applyCertificate(live, &plan); err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to update certificate for domain %q", plan.DomainName.ValueString()), err.Error())
		return
	}

	r.readIntoModel(live, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *LiveDomainCertificateResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state LiveDomainCertificateModel
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

	// Disable HTTPS.
	_, err = live.SetLiveDomainCertificate(&liveclient.SetLiveDomainCertificateRequest{
		DomainName:  strPtr(domainName),
		SSLProtocol: strPtr("off"),
		ForceSet:    strPtr("1"),
	})
	if err != nil && !strings.Contains(err.Error(), "InvalidDomain.NotFound") {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to disable certificate for domain %q", domainName), err.Error())
		return
	}

	// Disable HTTP/2.
	_ = batchDeleteConfig(live, domainName, "https_option")
}

func (r *LiveDomainCertificateResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("domain_name"), req, resp)
}

// applyCertificate calls SetLiveDomainCertificate and optionally configures HTTP/2.
func (r *LiveDomainCertificateResource) applyCertificate(live *liveclient.Client, m *LiveDomainCertificateModel) error {
	certReq := &liveclient.SetLiveDomainCertificateRequest{
		DomainName:  strPtr(m.DomainName.ValueString()),
		SSLProtocol: strPtr(m.SSLProtocol.ValueString()),
		ForceSet:    strPtr("1"),
	}
	if !m.SSLPub.IsNull() && !m.SSLPub.IsUnknown() {
		certReq.SSLPub = strPtr(m.SSLPub.ValueString())
	}
	if !m.SSLPri.IsNull() && !m.SSLPri.IsUnknown() {
		certReq.SSLPri = strPtr(m.SSLPri.ValueString())
	}
	if !m.CertName.IsNull() && !m.CertName.IsUnknown() {
		certReq.CertName = strPtr(m.CertName.ValueString())
	}
	if !m.CertType.IsNull() && !m.CertType.IsUnknown() {
		certReq.CertType = strPtr(m.CertType.ValueString())
	}
	if _, err := live.SetLiveDomainCertificate(certReq); err != nil {
		return err
	}

	// HTTP/2 is only meaningful when HTTPS is on.
	if m.SSLProtocol.ValueString() == "on" && !m.Http2.IsNull() && !m.Http2.IsUnknown() {
		if err := batchSetConfig(live, m.DomainName.ValueString(), "https_option", map[string]string{
			"http2": m.Http2.ValueString(),
		}); err != nil {
			return fmt.Errorf("set http2: %w", err)
		}
	}
	return nil
}

// readIntoModel populates computed fields from the API.
func (r *LiveDomainCertificateResource) readIntoModel(live *liveclient.Client, m *LiveDomainCertificateModel, diags interface {
	AddError(summary, detail string)
}) {
	domainName := m.DomainName.ValueString()

	certResp, err := live.DescribeLiveDomainCertificateInfo(&liveclient.DescribeLiveDomainCertificateInfoRequest{
		DomainName: strPtr(domainName),
	})
	if err != nil {
		diags.AddError(fmt.Sprintf("Failed to describe certificate for domain %q", domainName), err.Error())
		return
	}
	if certResp.Body != nil && certResp.Body.CertInfos != nil && len(certResp.Body.CertInfos.CertInfo) > 0 {
		info := certResp.Body.CertInfos.CertInfo[0]
		if info.SSLProtocol != nil {
			m.SSLProtocol = types.StringValue(*info.SSLProtocol)
		}
		if info.CertName != nil {
			m.CertName = types.StringValue(*info.CertName)
		}
		if info.CertType != nil {
			m.CertType = types.StringValue(*info.CertType)
		}
		if info.CertDomainName != nil {
			m.CertDomainName = types.StringValue(*info.CertDomainName)
		}
		if info.CertExpireTime != nil {
			m.CertExpireTime = types.StringValue(*info.CertExpireTime)
		}
		if info.SSLPub != nil {
			m.SSLPub = types.StringValue(*info.SSLPub)
		}
	}

	// Read HTTP/2 setting.
	args, err := describeFunctionArgs(live, domainName, "https_option")
	if err != nil {
		diags.AddError(fmt.Sprintf("Failed to describe https_option for domain %q", domainName), err.Error())
		return
	}
	if args != nil {
		if v, ok := args["http2"]; ok {
			m.Http2 = types.StringValue(v)
		}
	}
}
