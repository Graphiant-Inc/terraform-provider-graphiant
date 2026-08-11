data "graphiant_site" "hq" {
  id = 12345
}

output "hq_edge_count" {
  value = data.graphiant_site.hq.edge_count
}
