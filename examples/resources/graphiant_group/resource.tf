resource "graphiant_group" "example" {
  name        = "network-admins"
  description = "Full network configuration access"

  permissions {
    network_configuration          = "readWrite"
    monitoring_and_troubleshooting = "readWrite"
    reports                        = "read"
  }

  members = [
    graphiant_user.example.id,
  ]
}
