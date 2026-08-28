data "graphiant_alert_records" "current" {}

output "critical_alert_count" {
  value = length([for a in data.graphiant_alert_records.current.alerts : a if a.severity == "critical"])
}
