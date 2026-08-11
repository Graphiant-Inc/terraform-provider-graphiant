data "graphiant_groups" "all" {}

output "group_names" {
  value = [for g in data.graphiant_groups.all.groups : g.name]
}
