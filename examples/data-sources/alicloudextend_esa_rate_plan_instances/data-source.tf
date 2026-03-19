# Find the first online Premium (vipplan_intl / high) plan instance
# that still has remaining site quota.
data "alicloudextend_esa_rate_plan_instances" "premium" {
  # Defaults below are pre-set; explicit here for clarity.
  plan_name_en              = "vipplan_intl"
  plan_type                 = "high"
  status                    = "online"
  check_remaining_site_quota = true
}

output "premium_instance_id" {
  value = data.alicloudextend_esa_rate_plan_instances.premium.instance_id
}

output "all_premium_instances" {
  value = data.alicloudextend_esa_rate_plan_instances.premium.instances
}
