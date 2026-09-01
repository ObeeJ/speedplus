# ── R2 media bucket ───────────────────────────────────────────────────────────
resource "cloudflare_r2_bucket" "media" {
  account_id = var.cloudflare_account_id
  name       = var.r2_media_bucket_name

  # Location hint — valid values: WNAM, ENAM, WEUR, EEUR, APAC, OC
  # WEUR is closest to Nigeria via Cloudflare's edge network
  location = "WEUR"
}

# ── Custom domain for the media bucket ───────────────────────────────────────
# Serves media at media.speedplus.ng instead of the raw R2 URL.
resource "cloudflare_r2_bucket_custom_domain" "media" {
  account_id  = var.cloudflare_account_id
  bucket_name = cloudflare_r2_bucket.media.name
  domain      = "media.${var.zone_name}"
  zone_id     = var.cloudflare_zone_id
  enabled     = true
}

# ── NOTE: CORS & lifecycle ────────────────────────────────────────────────────
# The Cloudflare Terraform provider v4 does not yet expose R2 CORS rules or
# lifecycle policies as managed resources. Configure them via:
#
#   CORS (allow presigned PUT from app origins):
#     Dashboard → R2 → fourdat-media → Settings → CORS Policy
#     Or: curl -X PUT https://api.cloudflare.com/client/v4/accounts/<ACCOUNT_ID>/r2/buckets/<BUCKET>/cors
#
#   Lifecycle (expire tmp/ prefix after 1 day):
#     Dashboard → R2 → fourdat-media → Settings → Object Lifecycle
#
# Example CORS JSON to paste in the dashboard:
# [
#   {
#     "AllowedOrigins": [
#       "https://fourdat.com",
#       "https://ride.fourdat.com",
#       "https://merchant.fourdat.com",
#       "https://admin.fourdat.com"
#     ],
#     "AllowedMethods": ["GET", "PUT", "HEAD"],
#     "AllowedHeaders": ["Content-Type", "Content-Length", "Authorization"],
#     "ExposeHeaders": ["ETag"],
#     "MaxAgeSeconds": 3600
#   }
# ]
