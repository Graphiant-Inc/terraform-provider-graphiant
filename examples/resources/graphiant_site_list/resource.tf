resource "graphiant_site_list" "example" {
  name        = "west-coast-sites"
  description = "All West Coast sites"

  entries = [
    { site_id = graphiant_site.example.id },
  ]
}
