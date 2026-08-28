resource "graphiant_assurance_classified_application" "internal_erp" {
  app_name       = "internal-erp"
  ip_prefix_list = ["10.10.0.0/16"]
  port_list      = ["8443"]
  protocol_list  = ["tcp"]
}
