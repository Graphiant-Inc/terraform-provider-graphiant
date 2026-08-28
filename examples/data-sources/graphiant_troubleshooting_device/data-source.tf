data "graphiant_troubleshooting_device" "example" {
  device_id = 12345
}

output "device_health_status" {
  value = data.graphiant_troubleshooting_device.example.status
}
