resource "graphiant_user" "example" {
  email      = "jane.doe@example.com"
  first_name = "Jane"
  last_name  = "Doe"
  group_id   = graphiant_group.example.id
  time_zone  = "America/Los_Angeles"
}
