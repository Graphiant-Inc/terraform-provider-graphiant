resource "graphiant_custom_app" "example" {
  name        = "internal-erp"
  description = "Internal ERP system"
  ip_prefixes = ["10.10.0.0/16"]

  port_ranges = [
    { lower = 8443, upper = 8443 },
  ]
}
