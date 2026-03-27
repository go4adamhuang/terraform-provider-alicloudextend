# Step 1: Retrieve the TXT record verification content.
data "alicloudextend_live_domain_verify_content" "ingest" {
  domain_name = "push.example.com"
}

data "alicloudextend_live_domain_verify_content" "streaming" {
  domain_name = "play.example.com"
}

# Step 2: Create DNS TXT records using your DNS provider, e.g.:
#   _dnsauth.push.example.com  TXT  <data.alicloudextend_live_domain_verify_content.ingest.content>
#   _dnsauth.play.example.com  TXT  <data.alicloudextend_live_domain_verify_content.streaming.content>
# Then apply this configuration after the TXT records have propagated.

# Ingest (push) domain
resource "alicloudextend_live_domain" "ingest" {
  domain_name      = "push.example.com"
  live_domain_type = "liveEdge"
  region           = "cn-shanghai"
  scope            = "domestic"
}

# Streaming (play) domain
resource "alicloudextend_live_domain" "streaming" {
  domain_name      = "play.example.com"
  live_domain_type = "liveVideo"
  region           = "cn-shanghai"
  scope            = "domestic"
}

output "ingest_cname" {
  value       = alicloudextend_live_domain.ingest.cname
  description = "Point push.example.com CNAME to this value."
}

output "streaming_cname" {
  value       = alicloudextend_live_domain.streaming.cname
  description = "Point play.example.com CNAME to this value."
}
