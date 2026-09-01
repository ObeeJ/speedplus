output "pages_urls" {
  description = "Cloudflare Pages default *.pages.dev URLs for each app"
  value = {
    for k, v in cloudflare_pages_project.apps :
    k => "https://${v.name}.pages.dev"
  }
}

output "custom_domains" {
  description = "Production custom domains"
  value = {
    customer = "https://fourdat.com"
    driver   = "https://ride.fourdat.com"
    merchant = "https://merchant.fourdat.com"
    admin    = "https://admin.fourdat.com"
  }
}

output "r2_bucket_name" {
  description = "R2 media bucket name"
  value       = cloudflare_r2_bucket.media.name
}

output "r2_media_domain" {
  description = "Public URL for the R2 media bucket custom domain"
  value       = "https://media.${var.zone_name}"
}
