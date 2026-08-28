data "graphiant_assurance_dnsproxy_entries" "all" {}

output "dnsproxy_entry_names" {
  value = [for e in data.graphiant_assurance_dnsproxy_entries.all.entries : e.name]
}
