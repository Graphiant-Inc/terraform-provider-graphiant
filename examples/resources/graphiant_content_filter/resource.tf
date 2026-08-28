resource "graphiant_content_filter" "example" {
  name          = "block-social-media"
  use_all_sites = true

  rules = [
    { domain_category_id = 42 },
  ]
}
