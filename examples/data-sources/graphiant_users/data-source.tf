data "graphiant_users" "all" {}

output "user_emails" {
  value = [for u in data.graphiant_users.all.users : u.email]
}
