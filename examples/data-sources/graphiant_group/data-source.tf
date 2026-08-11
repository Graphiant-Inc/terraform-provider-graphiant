data "graphiant_group" "network_admins" {
  id = "grp-abc123"
}

output "network_admins_permissions" {
  value = data.graphiant_group.network_admins.permissions
}
