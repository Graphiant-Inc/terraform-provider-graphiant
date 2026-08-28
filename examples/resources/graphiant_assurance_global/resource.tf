resource "graphiant_assurance_global" "critical_apps" {
  name          = "critical-apps"
  use_all_sites = true

  lan_names = ["corp-lan"]

  apps = [
    { custom_app_id = graphiant_custom_app.example.id },
  ]
}
