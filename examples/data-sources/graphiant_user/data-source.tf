data "graphiant_user" "jane" {
  id = "jane@example.com"
}

output "jane_group_verified" {
  value = data.graphiant_user.jane.verified
}
