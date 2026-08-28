resource "graphiant_extranet" "shared_services" {
  name        = "shared-services"
  description = "Share the shared-services segment with branch sites"

  shared_segment  = 100
  target_segments = [200, 201]

  auto = {
    auto_propagate = true
  }

  branches = {
    sites = [graphiant_site.example.id]
  }
}
