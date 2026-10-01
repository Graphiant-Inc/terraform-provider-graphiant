data "graphiant_playbook_templates" "ntp" {
  bundle_key  = "system_bundle"
  module_keys = ["ntp"]
}
