resource "graphiant_software_rollout" "canary" {
  action  = "upgrade"
  name    = "canary-rollout"
  release = "26.8.0"

  device_ids = [12345, 12346]

  trigger_now = true
}
