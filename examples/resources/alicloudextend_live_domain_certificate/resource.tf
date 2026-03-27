resource "alicloudextend_live_domain_certificate" "example" {
  domain_name  = "play.example.com"
  ssl_protocol = "on"
  cert_type    = "upload"
  ssl_pub      = file("path/to/cert.pem")
  ssl_pri      = file("path/to/key.pem")
  cert_name    = "my-live-cert"
  http2        = "on"
}
