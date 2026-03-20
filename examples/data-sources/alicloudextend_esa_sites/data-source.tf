data "alicloudextend_esa_sites" "existing" {}

output "sites" {
  value = data.alicloudextend_esa_sites.existing.sites
}
