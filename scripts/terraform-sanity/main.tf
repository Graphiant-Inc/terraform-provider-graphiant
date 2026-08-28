# Minimal root module for scripts/terraform-sanity.sh: exercises the real
# provider binary through the real Terraform CLI (via a dev override), as
# opposed to cmd/sanity, which talks to the SDK directly with no Terraform
# involved at all. Credentials/host are read from the environment
# (GRAPHIANT_ACCESS_TOKEN, or GRAPHIANT_USERNAME + GRAPHIANT_PASSWORD, and
# optionally GRAPHIANT_API_HOST/GRAPHIANT_HOST) — the provider block below
# intentionally sets nothing, so provider.go falls back to that resolution.

terraform {
  required_providers {
    graphiant = {
      source = "Graphiant-Inc/graphiant"
    }
  }
}

provider "graphiant" {}

data "graphiant_edges" "sanity" {}

output "edge_count" {
  value = length(data.graphiant_edges.sanity.edges)
}

output "edges" {
  value = [
    for e in data.graphiant_edges.sanity.edges : {
      device_id = e.device_id
      hostname  = e.hostname
      status    = e.status
      role      = e.role
      site      = e.site
    }
  ]
}
