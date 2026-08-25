data "graphiant_app_lists" "all" {}

output "app_list_names" {
  value = [for al in data.graphiant_app_lists.all.app_lists : al.name]
}
