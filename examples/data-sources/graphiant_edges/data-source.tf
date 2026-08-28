data "graphiant_edges" "all" {}

output "online_edge_count" {
  value = length([for e in data.graphiant_edges.all.edges : e if e.status == "active"])
}
