resource "graphiant_gateway" "ipsec" {
  region_id = 1
  vrf_id    = 100
  speed     = "1G"

  ipsec_gateway = {
    destination_address      = "203.0.113.10"
    ike_initiator            = true
    vpn_profile              = "default"
    remote_ike_peer_identity = "peer.example.com"

    tunnel1 = {
      inside_ipv4_cidr = "169.254.0.0/30"
      psk              = var.ipsec_psk
    }
  }
}
