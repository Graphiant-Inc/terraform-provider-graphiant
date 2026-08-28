resource "graphiant_app_list" "example" {
  name        = "internal-apps"
  description = "Internal business applications"

  apps = [
    { id = graphiant_custom_app.example.id, type = "custom" },
  ]
}
