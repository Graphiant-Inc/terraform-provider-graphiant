data "graphiant_domain_categories" "all" {}

output "social_media_category_id" {
  value = [for c in data.graphiant_domain_categories.all.domain_categories : c.id if c.name == "Social Media"][0]
}
