terraform {
  required_providers {
    cachix = {
      source  = "takeokunn/cachix"
      version = "~> 0.1"
    }
  }
}

# Configure the Cachix provider
# Option 1: Set CACHIX_AUTH_TOKEN environment variable and use empty provider block
# Option 2: Provide auth_token explicitly (shown below)
provider "cachix" {
  # auth_token = var.cachix_token  # Or use CACHIX_AUTH_TOKEN env var
  # api_host   = "https://app.cachix.org/api/v1"  # Optional, defaults to this value
}
