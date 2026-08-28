resource "graphiant_alert_integration" "oncall_webhook" {
  enterprise       = 12345
  integration_type = "webhook"
  nick_name        = "oncall-webhook"
  is_active        = true

  details = {
    webhook_url = "https://hooks.example.com/oncall"
  }
}
