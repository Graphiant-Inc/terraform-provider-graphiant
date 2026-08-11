data "graphiant_devices" "all" {}

output "device_count" {
  value = length(data.graphiant_devices.all.devices)
}
