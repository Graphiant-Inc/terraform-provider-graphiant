data "graphiant_troubleshooting_site" "example" {
  site_id = graphiant_site.example.id
}

output "site_status" {
  value = data.graphiant_troubleshooting_site.example.site_status
}
