resource "alicloudextend_live_domain_access_control" "example" {
  domain_name = "play.example.com"
  rtmp_block  = "on"
  hls_block   = "on"
}
