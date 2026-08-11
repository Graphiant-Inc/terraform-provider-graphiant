resource "graphiant_user" "jane" {
  email      = "jane@example.com"
  first_name = "Jane"
  last_name  = "Doe"
  group_id   = graphiant_group.network_admins.id
}
