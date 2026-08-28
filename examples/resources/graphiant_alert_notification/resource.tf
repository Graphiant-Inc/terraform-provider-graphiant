resource "graphiant_alert_notification" "oncall" {
  notification_name = "oncall-critical"
  rule_id_list      = ["rule-123"]

  enabled        = true
  duration       = "5m"
  frequency      = 15
  recipient_list = ["oncall@example.com"]
}
