# Each resource manages one HTTP response header.
resource "alicloudextend_live_domain_http_header" "cors" {
  domain_name = "play.example.com"
  key         = "Access-Control-Allow-Origin"
  value       = "*"
}

resource "alicloudextend_live_domain_http_header" "cache_control" {
  domain_name = "play.example.com"
  key         = "Cache-Control"
  value       = "no-cache"
}
