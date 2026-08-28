resource "graphiant_b2b_producer_service" "partner_peering" {
  service_name = "partner-peering-service"
  service_type = "peering_service"

  policy = {
    description         = "Peering service for partner network"
    service_lan_segment = 100

    sites = [
      { sites = [graphiant_site.example.id] },
    ]

    prefix_tags = [
      { prefix = "10.40.0.0/16", tag = "shared" },
    ]

    nat_translation_mode = {
      peer_to_peer = [
        { prefix = "10.40.0.0/16" },
      ]
    }
  }
}
