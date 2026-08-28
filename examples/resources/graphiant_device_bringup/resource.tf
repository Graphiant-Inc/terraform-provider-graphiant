# Action-shaped: triggers a bulk bringup status transition. There is no
# "un-bringup" endpoint, so `terraform destroy` only removes this from state.
resource "graphiant_device_bringup" "activate" {
  device_ids = [12345, 12346]
  status     = "activate"
}
