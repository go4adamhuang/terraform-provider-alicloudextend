---
page_title: "alicloudextend_esa_rate_plan_instances Data Source - alicloudextend"
subcategory: "ESA (Edge Security Acceleration)"
description: |-
  Lists ESA rate plan instances. Defaults to finding the first online Premium (vipplan_intl / high) instance with remaining site quota.
---

# alicloudextend_esa_rate_plan_instances

Lists ESA rate plan instances via the `ListUserRatePlanInstances` API.

The primary use case is to look up the **instance ID of a Premium plan** (`vipplan_intl` / `high`) that still has remaining site quota, so it can be passed directly to `alicloud_esa_site`.

## Example Usage

### Look up an available Premium instance (default filters)

```terraform
data "alicloudextend_esa_rate_plan_instances" "premium" {}

resource "alicloud_esa_site" "example" {
  site_name   = "example.com"
  instance_id = data.alicloudextend_esa_rate_plan_instances.premium.instance_id
  coverage    = "overseas"
  access_type = "NS"
}
```

### Explicit filters

```terraform
data "alicloudextend_esa_rate_plan_instances" "premium" {
  plan_name_en               = "vipplan_intl"
  plan_type                  = "high"
  status                     = "online"
  check_remaining_site_quota = true
}

output "instance_id" {
  value = data.alicloudextend_esa_rate_plan_instances.premium.instance_id
}
```

### List all instances regardless of quota

```terraform
data "alicloudextend_esa_rate_plan_instances" "all" {
  check_remaining_site_quota = false
}

output "instances" {
  value = data.alicloudextend_esa_rate_plan_instances.all.instances
}
```

## Schema

### Optional

- `plan_name_en` (String) Filter by plan name. Defaults to `"vipplan_intl"`.
- `plan_type` (String) Filter by plan type. Defaults to `"high"`. Valid values: `high`, `normal`, `enterprise`.
- `status` (String) Filter by plan status. Defaults to `"online"`. Valid values: `online`, `offline`, `disable`, `overdue`.
- `check_remaining_site_quota` (Boolean) When `true`, only returns instances that still have remaining site quota. Defaults to `true`.

### Read-Only

- `instance_id` (String) The `instance_id` of the **first** matching plan instance. Use this directly in `alicloud_esa_site.instance_id`.
- `instances` (List of Object) All matching plan instances. Each object contains:
  - `instance_id` (String) Plan instance ID (e.g. `sp-xxxx`).
  - `plan_name` (String) Plan name.
  - `plan_type` (String) Plan type.
  - `status` (String) Plan status.
  - `site_quota` (String) Maximum number of websites that can be associated with this plan.
  - `expire_time` (String) Plan expiration time.
  - `coverages` (String) Service locations covered by the plan (e.g. `domestic,overseas`).

## Notes

- Results are fetched with automatic pagination (page size 100). All matching instances across all pages are returned in `instances`.
- `instance_id` is set to the first element of `instances`. If multiple Premium plans are active, pin the specific one by iterating `instances` rather than relying on `instance_id`.
- The ESA API endpoint used is `esa.<region>.aliyuncs.com`. If no `region` is configured in the provider, it defaults to `esa.cn-hangzhou.aliyuncs.com`.
