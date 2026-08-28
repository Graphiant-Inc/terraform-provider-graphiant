data "graphiant_regions" "all" {}

output "available_region_ids" {
  value = [for r in data.graphiant_regions.all.regions : r.id if !r.unavailable]
}
