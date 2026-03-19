---
page_title: "Provider: AliCloud Extend"
description: |-
  The AliCloud Extend provider supplements the official AliCloud provider with additional resources and data sources.
---

# AliCloud Extend Provider

The **AliCloud Extend** provider is a supplemental provider for [Alibaba Cloud](https://www.alibabacloud.com), adding resources absent from the [official provider](https://registry.terraform.io/providers/aliyun/alicloud/latest) and rewriting certain resource logic for internal use cases.

Built on [terraform-plugin-framework](https://github.com/hashicorp/terraform-plugin-framework).

## Authentication

Credentials can be supplied in three ways (highest priority first):

1. **Explicit provider block** — `access_key_id`, `access_key_secret`, and `region` attributes.
2. **Alibaba Cloud CLI profile** — `profile` attribute reads from `~/.aliyun/config.json`.
3. **Environment variables** — `ALICLOUD_ACCESS_KEY_ID`, `ALICLOUD_ACCESS_KEY_SECRET`, `ALICLOUD_REGION_ID`.

Using a `profile` is recommended so credentials are never committed to code.

## Example Usage

```terraform
terraform {
  required_providers {
    alicloudextend = {
      source  = "go4adamhuang/alicloudextend"
      version = "~> 1.0"
    }
  }
}

# Use Alibaba Cloud CLI profile (recommended)
provider "alicloudextend" {
  profile = "default"
}
```

```terraform
# Explicit credentials
provider "alicloudextend" {
  access_key_id     = var.access_key_id
  access_key_secret = var.access_key_secret
  region            = "cn-hangzhou"
}
```

```terraform
# Environment variables — no provider block needed
# export ALICLOUD_ACCESS_KEY_ID=...
# export ALICLOUD_ACCESS_KEY_SECRET=...
# export ALICLOUD_REGION_ID=cn-hangzhou
provider "alicloudextend" {}
```

## Profile Format

The `profile` attribute resolves credentials from the standard Alibaba Cloud CLI config file at `~/.aliyun/config.json`. The relevant fields are:

```json
{
  "profiles": [
    {
      "name": "default",
      "mode": "AK",
      "access_key_id": "LTAI...",
      "access_key_secret": "...",
      "region_id": "cn-hangzhou"
    }
  ]
}
```

Run `aliyun configure` to create or update profiles. The `profile` attribute (or `ALICLOUD_PROFILE` env var) selects which entry to use.

## Schema

### Optional

- `access_key_id` (String) Alibaba Cloud access key ID. Priority: explicit > profile > `ALICLOUD_ACCESS_KEY_ID` env var.
- `access_key_secret` (String, Sensitive) Alibaba Cloud access key secret. Priority: explicit > profile > `ALICLOUD_ACCESS_KEY_SECRET` env var.
- `region` (String) Alibaba Cloud region ID (e.g. `cn-hangzhou`). Priority: explicit > profile > `ALICLOUD_REGION_ID` env var.
- `profile` (String) Alibaba Cloud CLI profile name. Loads credentials from `~/.aliyun/config.json`. Falls back to `ALICLOUD_PROFILE` env var.
