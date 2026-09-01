terraform {
  required_version = ">= 1.6"

  required_providers {
    cloudflare = {
      source  = "cloudflare/cloudflare"
      version = "~> 4.0"
    }
  }

  # State stored in Cloudflare R2 — no AWS account needed.
  # Init with:
  #   terraform init \
  #     -backend-config="endpoint=https://<ACCOUNT_ID>.r2.cloudflarestorage.com" \
  #     -backend-config="access_key=<R2_ACCESS_KEY_ID>" \
  #     -backend-config="secret_key=<R2_SECRET_ACCESS_KEY>"
  backend "s3" {
    bucket                      = "fourdat-tf-state"
    key                         = "cloudflare/terraform.tfstate"
    region                      = "auto"
    skip_credentials_validation = true
    skip_metadata_api_check     = true
    skip_region_validation      = true
    force_path_style            = true
  }
}

provider "cloudflare" {
  api_token = var.cloudflare_api_token
}
