# Action-shaped: drives the hardware-return workflow. Keyed by serial number,
# not device id.
resource "graphiant_device_decommission" "retire" {
  device_serials = ["SN-0001", "SN-0002"]

  approve = true
  clear   = false
}
