data "graphiant_custom_apps" "all" {}

output "custom_app_names" {
  value = [for a in data.graphiant_custom_apps.all.custom_apps : a.name]
}
