data "graphiant_assurance_flex_algos" "all" {}

output "flex_algo_names" {
  value = [for f in data.graphiant_assurance_flex_algos.all.flex_algos : f.name]
}
