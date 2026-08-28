data "graphiant_alert_rules" "all" {}

output "disabled_rules" {
  value = [for r in data.graphiant_alert_rules.all.rules : r.rule_name if !r.enabled]
}
