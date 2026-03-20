# Terraform Provider: AliCloud Extend

Extended Terraform provider for Alibaba Cloud — covering resources not provided by, or rewritten from, the official [`aliyun/alicloud`](https://registry.terraform.io/providers/aliyun/alicloud/latest) provider.

Built with [Terraform Plugin Framework](https://github.com/hashicorp/terraform-plugin-framework).

## Usage

```terraform
terraform {
  required_providers {
    alicloudextend = {
      source  = "go4adamhuang/alicloudextend"
      version = "~> 1.0"
    }
  }
}

provider "alicloudextend" {
  profile = "default"
}
```

## Authentication

Credentials are resolved in this order (lowest → highest priority):

| Method | Details |
|---|---|
| Environment variables | `ALICLOUD_ACCESS_KEY_ID`, `ALICLOUD_ACCESS_KEY_SECRET`, `ALICLOUD_REGION_ID` |
| CLI profile | `profile` attribute, reads from `~/.aliyun/config.json` |
| Explicit block | `access_key_id`, `access_key_secret`, `region` in provider block |

Using a profile is recommended so credentials are never committed to code.

## Resources & Data Sources

### ESA (Edge Security Acceleration)

| Type | Name | Description |
|---|---|---|
| Resource | `alicloudextend_esa_https_basic_configuration` | Manages HTTPS basic configuration for an ESA site. Uses upsert semantics to avoid `ConfExceedLimit` errors. |
| Data Source | `alicloudextend_esa_rate_plan_instances` | Lists ESA rate plan instances, with optional filtering by plan name, type, and status. |
| Data Source | `alicloudextend_esa_sites` | Lists all ESA sites under the account. |

## Development

### Requirements

- [Go](https://golang.org/) 1.22+
- [Terraform](https://www.terraform.io/) 1.0+

### Commands

```bash
make build      # compile provider binary
make install    # build + install into ~/.terraform.d/plugins/ for local testing
make test       # unit tests
make testacc    # acceptance tests (requires real AliCloud credentials)
make fmt        # go fmt + goimports
make lint       # golangci-lint
make docs       # generate docs/ via tfplugindocs
```

### Local Testing

```bash
make install
```

Then reference the local provider in your Terraform config:

```terraform
terraform {
  required_providers {
    alicloudextend = {
      source  = "go4adamhuang/alicloudextend"
      version = "~> 1.0"
    }
  }
}
```

And configure `~/.terraformrc`:

```hcl
provider_installation {
  dev_overrides {
    "go4adamhuang/alicloudextend" = "/path/to/terraform-provider-alicloudextend"
  }
  direct {}
}
```
