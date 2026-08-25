resource "graphiant_site_list" "west_coast" {
  name        = "west-coast-sites"
  description = "Sites used as the scope for west-coast content filtering"

  entries = [
    { site_id = graphiant_site.hq.id },
    { tag = { level_zero = 1, level_one = 2 } },
  ]
}
