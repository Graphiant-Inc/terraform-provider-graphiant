# Only maintenance_mode and (for edge devices) the *_enabled toggles round-trip
# from the API on read; region/description/local_web_server_password/replace
# are write-only. See the resource description for what this does NOT cover
# (BGP, interfaces, NAT/security/traffic policy, site-to-site VPN, etc.).
resource "graphiant_device_config" "edge_maintenance" {
  device_id   = 12345
  device_type = "edge"

  maintenance_mode = true
}
