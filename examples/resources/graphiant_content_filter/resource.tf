resource "graphiant_content_filter" "block_social_media" {
  name          = "block-social-media"
  use_all_sites = true

  rules = [
    {
      domain_category_id  = 42
      exception_wildcards = ["*.corp-approved-app.example.com"]
    },
  ]
}
