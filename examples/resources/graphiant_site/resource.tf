resource "graphiant_site" "example" {
  name  = "sf-hq"
  notes = "San Francisco headquarters"

  location {
    address_line1 = "123 Market St"
    city          = "San Francisco"
    state_code    = "CA"
    country_code  = "US"
    latitude      = 37.7749
    longitude     = -122.4194
  }

  route_tag {
    level_zero = "prod"
  }
}
