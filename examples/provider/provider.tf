terraform {
  required_providers {
    graphiant = {
      source = "Graphiant-Inc/graphiant"
    }
  }
}

# Credentials can also be supplied via GRAPHIANT_ACCESS_TOKEN, or
# GRAPHIANT_USERNAME + GRAPHIANT_PASSWORD, and the host via GRAPHIANT_API_HOST.
provider "graphiant" {
  host         = "https://api.graphiant.com"
  access_token = var.graphiant_access_token
}
