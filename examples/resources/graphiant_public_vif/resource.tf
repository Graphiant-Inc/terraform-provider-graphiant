resource "graphiant_public_vif" "example" {
  service_name     = "public-vif-1"
  lan_segment_id   = 100
  region_id        = 1
  storage_provider = "aws"

  consumer_lan_segments = {
    "200" = {
      consumer_prefixes = ["10.20.0.0/16"]
    }
  }

  gateway_bgp_neighbors = {
    "1" = {
      enabled        = true
      peer_asn       = 65001
      remote_address = "203.0.113.1"
      local_address  = "203.0.113.2"
    }
  }

  nat_prefix_strategy = {
    decentralized = {
      prefixes = {
        "1" = "10.30.0.0/16"
      }
    }
  }
}
