data "graphiant_app_list" "productivity_apps" {
  id = 12345
}

output "productivity_apps_count" {
  value = length(data.graphiant_app_list.productivity_apps.apps)
}
