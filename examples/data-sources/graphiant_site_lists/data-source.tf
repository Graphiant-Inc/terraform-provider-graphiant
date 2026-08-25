data "graphiant_site_lists" "all" {}

output "site_list_names" {
  value = [for sl in data.graphiant_site_lists.all.site_lists : sl.name]
}
