# Discover bundle and module keys with the graphiant_playbook_bundles and
# graphiant_playbook_module_slots data sources, and starter YAML with
# graphiant_playbook_templates.
resource "graphiant_playbook_config" "ntp" {
  name        = "branch-ntp"
  description = "Standard NTP servers for branch edges"
  bundle_key  = "system_bundle"

  files = [{
    module_key = "ntp"
    filename   = "ntp.yaml"
    content    = file("${path.module}/configs/ntp.yaml")
  }]
}
