resource "graphiant_enterprise" "customer" {
  account_type = "enterprise"
  company_name = "Acme Corp"

  admin_email      = "admin@acme.example"
  admin_first_name = "Jane"
  admin_last_name  = "Doe"

  credit_limit = 1000

  enterprise_contract = {
    contracted_credits = 500
    expiration_date = {
      month = 12
      year  = 2026
    }
  }
}
