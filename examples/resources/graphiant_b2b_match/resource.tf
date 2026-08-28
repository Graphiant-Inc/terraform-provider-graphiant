resource "graphiant_b2b_match" "partner_match" {
  customer_id = graphiant_b2b_customer.partner.id

  match = {
    service_id        = graphiant_b2b_producer_service.partner_peering.id
    lan_segment       = 100
    consumer_prefixes = ["10.50.0.0/16"]
  }
}
