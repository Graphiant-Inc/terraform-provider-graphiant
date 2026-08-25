resource "graphiant_custom_app" "internal_wiki" {
  name        = "internal-wiki"
  description = "Internal wiki, matched by URL and port"
  url         = "wiki.internal.example.com"
  ip_protocol = "tcp"

  port_ranges = [
    { lower = 443, upper = 443 },
  ]
}
