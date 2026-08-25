data "graphiant_site_list" "west_coast" {
  id = 12345
}

output "west_coast_member_count" {
  value = length(data.graphiant_site_list.west_coast.entries)
}
