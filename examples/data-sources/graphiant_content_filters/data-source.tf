data "graphiant_content_filters" "all" {}

output "content_filter_names" {
  value = [for f in data.graphiant_content_filters.all.content_filters : f.name]
}
