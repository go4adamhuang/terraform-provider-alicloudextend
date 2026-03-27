# Step 1: Get the TXT record value needed to prove domain ownership.
data "alicloudextend_live_domain_verify_content" "example" {
  domain_name = "live.example.com"
}

# Step 2: Add a DNS TXT record with the returned content.
#   Host:  _dnsauth.live.example.com
#   Value: data.alicloudextend_live_domain_verify_content.example.content
output "verify_txt_value" {
  value = data.alicloudextend_live_domain_verify_content.example.content
}
