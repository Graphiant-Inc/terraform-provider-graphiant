# No import support: the API has no get-by-consumer-id endpoint (see the
# resource description), so this resource is created via Terraform and
# managed for the lifetime of the config, but cannot be imported.
resource "graphiant_b2b_consumer" "partner_accept" {
  customer_id = graphiant_b2b_customer.partner.id
  match_id    = graphiant_b2b_match.partner_match.id
  service_id  = graphiant_b2b_producer_service.partner_peering.id

  policy = {
    consumer_lan_segments = {
      "200" = {
        consumer_prefixes = ["10.50.0.0/16"]
      }
    }
  }
}
