resource "alicloudextend_live_domain_certificate" "example" {
  domain_name  = "play.example.com"
  ssl_protocol = "on"
  cert_type    = "upload"
  ssl_pub      = file("path/to/cert.pem")
  ssl_pri      = file("path/to/key.pem")
}

resource "alicloudextend_live_domain_https_option" "example" {
  domain_name = "play.example.com"
  http2       = "on"

  depends_on = [alicloudextend_live_domain_certificate.example]
}
