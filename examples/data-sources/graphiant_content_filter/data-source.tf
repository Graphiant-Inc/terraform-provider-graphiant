data "graphiant_content_filter" "block_social_media" {
  id = 12345
}

output "block_social_media_rule_count" {
  value = length(data.graphiant_content_filter.block_social_media.rules)
}
