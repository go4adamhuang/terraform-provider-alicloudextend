# Step 1: Retrieve the TXT record verification content.
data "alicloudextend_live_domain_verify_content" "ingest" {
  domain_name = "push.example.com"
}

data "alicloudextend_live_domain_verify_content" "streaming" {
  domain_name = "play.example.com"
}

# Step 2: Create DNS TXT records using your DNS provider:
#   _dnsauth.push.example.com  TXT  <data.alicloudextend_live_domain_verify_content.ingest.content>
#   _dnsauth.play.example.com  TXT  <data.alicloudextend_live_domain_verify_content.streaming.content>
# Apply this configuration after TXT records have propagated.

# Ingest (push) domain
resource "alicloudextend_live_domain" "ingest" {
  domain_name = "push.example.com"
  domain_type = "liveEdge"
  region      = "cn-shanghai"
  scope       = "domestic"
  status      = "online"

  tags = {
    env = "production"
  }
}

# Streaming (play) domain
resource "alicloudextend_live_domain" "streaming" {
  domain_name = "play.example.com"
  domain_type = "liveVideo"
  region      = "cn-shanghai"
  scope       = "domestic"
  status      = "online"

  tags = {
    env = "production"
  }
}

# Step 3: Use the CNAME to create DNS records pointing to AliCloud CDN.
output "ingest_cname" {
  value       = alicloudextend_live_domain.ingest.cname
  description = "Point push.example.com CNAME to this value."
}

output "streaming_cname" {
  value       = alicloudextend_live_domain.streaming.cname
  description = "Point play.example.com CNAME to this value."
}
