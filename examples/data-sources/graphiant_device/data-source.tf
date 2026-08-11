data "graphiant_device" "edge1" {
  id = 67890
}

output "edge1_status" {
  value = data.graphiant_device.edge1.status
}
