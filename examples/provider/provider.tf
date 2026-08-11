terraform {
  required_providers {
    graphiant = {
      source = "Graphiant-Inc/graphiant"
    }
  }
}

# host, access_token, username, and password can all be set via environment
# variables instead (GRAPHIANT_API_HOST, GRAPHIANT_ACCESS_TOKEN,
# GRAPHIANT_USERNAME, GRAPHIANT_PASSWORD) — recommended for anything other
# than a static access_token sourced from a variable, as shown here.
provider "graphiant" {
  host         = "https://api.graphiant.com"
  access_token = var.graphiant_access_token
}

variable "graphiant_access_token" {
  description = "Bearer token issued by the Graphiant portal/CLI."
  type        = string
  sensitive   = true
}
