resource "graphiant_site" "hq" {
  name  = "Headquarters"
  notes = "Managed by Terraform"

  location = {
    address_line1 = "123 Main St"
    city          = "San Jose"
    state_code    = "CA"
    country_code  = "US"
  }
}
