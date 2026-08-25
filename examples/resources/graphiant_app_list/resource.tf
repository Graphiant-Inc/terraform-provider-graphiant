resource "graphiant_app_list" "productivity_apps" {
  name        = "productivity-apps"
  description = "Apps allowed under the productivity traffic policy"

  apps = [
    { id = 101, type = "graphiant" },
    { id = graphiant_custom_app.internal_wiki.id, type = "custom" },
  ]
}
