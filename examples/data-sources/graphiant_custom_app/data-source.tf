data "graphiant_custom_app" "internal_wiki" {
  id = 12345
}

output "internal_wiki_policy_references" {
  value = data.graphiant_custom_app.internal_wiki.policy_reference_count
}
