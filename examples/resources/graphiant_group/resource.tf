resource "graphiant_group" "network_admins" {
  name        = "network-admins"
  description = "Full network configuration access"

  permissions = {
    network_configuration          = "write"
    monitoring_and_troubleshooting = "write"
    insights                       = "read"
  }
}
