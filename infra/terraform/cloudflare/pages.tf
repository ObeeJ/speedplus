# ── Shared env vars injected into every app ──────────────────────────────────
locals {
  shared_env = {
    NEXT_PUBLIC_API_URL = var.api_url
  }

  apps = {
    customer = {
      project_name = "fourdat-customer"
      root_dir     = "apps/customer"
      domain       = "fourdat.com"
      extra_env    = var.customer_app_env
    }
    driver = {
      project_name = "fourdat-driver"
      root_dir     = "apps/driver"
      domain       = "ride.fourdat.com"
      extra_env    = var.driver_app_env
    }
    merchant = {
      project_name = "fourdat-merchant"
      root_dir     = "apps/merchant"
      domain       = "merchant.fourdat.com"
      extra_env    = var.merchant_app_env
    }
    admin = {
      project_name = "fourdat-admin"
      root_dir     = "apps/admin"
      domain       = "admin.fourdat.com"
      extra_env    = var.admin_app_env
    }
  }
}

# ── Cloudflare Pages projects ─────────────────────────────────────────────────
resource "cloudflare_pages_project" "apps" {
  for_each   = local.apps
  account_id = var.cloudflare_account_id
  name       = each.value.project_name

  production_branch = "main"

  source {
    type = "github"
    config {
      owner                         = var.github_owner
      repo_name                     = var.github_repo
      production_branch             = "main"
      pr_comments_enabled           = true
      deployments_enabled           = true
      production_deployment_enabled = true
    }
  }

  build_config {
    build_command   = "pnpm turbo build --filter=@speedplus/${each.key}"
    destination_dir = "apps/${each.key}/.next"
    root_dir        = ""
  }

  deployment_configs {
    production {
      environment_variables = merge(local.shared_env, each.value.extra_env)
      compatibility_date    = "2024-09-23"
      compatibility_flags   = ["nodejs_compat"]
    }

    preview {
      environment_variables = merge(local.shared_env, each.value.extra_env)
      compatibility_date    = "2024-09-23"
      compatibility_flags   = ["nodejs_compat"]
    }
  }
}

# ── Custom domains ────────────────────────────────────────────────────────────
resource "cloudflare_pages_domain" "apps" {
  for_each     = local.apps
  account_id   = var.cloudflare_account_id
  project_name = cloudflare_pages_project.apps[each.key].name
  domain       = each.value.domain
}

# ── DNS: apex → customer app ──────────────────────────────────────────────────
resource "cloudflare_record" "customer_apex" {
  zone_id = var.cloudflare_zone_id
  name    = "@"
  type    = "CNAME"
  content = cloudflare_pages_project.apps["customer"].domains[0]
  proxied = true

  depends_on = [cloudflare_pages_domain.apps]
}

# ── DNS: subdomains → their respective apps ───────────────────────────────────
resource "cloudflare_record" "subdomains" {
  for_each = {
    ride     = "driver"
    merchant = "merchant"
    admin    = "admin"
  }

  zone_id = var.cloudflare_zone_id
  name    = each.key
  type    = "CNAME"
  content = cloudflare_pages_project.apps[each.value].domains[0]
  proxied = true

  depends_on = [cloudflare_pages_domain.apps]
}

# ── DNS: api subdomain → Go API host ─────────────────────────────────────────
resource "cloudflare_record" "api" {
  zone_id = var.cloudflare_zone_id
  name    = "api"
  type    = "CNAME"
  content = var.api_hostname
  proxied = true
}
