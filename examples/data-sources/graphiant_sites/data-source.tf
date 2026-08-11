data "graphiant_sites" "all" {}

output "site_names" {
  value = [for s in data.graphiant_sites.all.sites : s.name]
}
