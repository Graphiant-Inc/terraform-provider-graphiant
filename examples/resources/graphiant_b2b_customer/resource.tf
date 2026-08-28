resource "graphiant_b2b_customer" "partner" {
  name = "Partner Co"
  type = "non-graphiant"

  invite = {
    admin_emails            = ["admin@partner.example"]
    maximum_number_of_sites = 5
  }
}
