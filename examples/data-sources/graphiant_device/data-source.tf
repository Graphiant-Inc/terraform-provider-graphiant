data "graphiant_device" "example" {
  id = 12345
}

output "device_hostname" {
  value = data.graphiant_device.example.hostname
}
