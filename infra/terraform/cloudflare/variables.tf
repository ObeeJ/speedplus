# ── Cloudflare ────────────────────────────────────────────────────────────────
variable "cloudflare_api_token" {
  description = "Cloudflare API token with Pages, DNS, and R2 permissions"
  type        = string
  sensitive   = true
}

variable "cloudflare_account_id" {
  description = "Cloudflare account ID"
  type        = string
}

variable "cloudflare_zone_id" {
  description = "Cloudflare zone ID for fourdat.com"
  type        = string
}

variable "zone_name" {
  description = "Root domain name"
  type        = string
  default     = "fourdat.com"
}

# ── GitHub ────────────────────────────────────────────────────────────────────
variable "github_owner" {
  description = "GitHub org or username that owns the monorepo"
  type        = string
}

variable "github_repo" {
  description = "Repository name (without owner prefix)"
  type        = string
  default     = "speedplus"
}

# ── API ───────────────────────────────────────────────────────────────────────
variable "api_url" {
  description = "Full public URL of the Go API"
  type        = string
  default     = "https://api.fourdat.com"
}

variable "api_hostname" {
  description = "Hostname of the Go API server for the CNAME record e.g. your-vps.example.com"
  type        = string
}

# ── R2 ────────────────────────────────────────────────────────────────────────
variable "r2_media_bucket_name" {
  description = "Name of the R2 media bucket"
  type        = string
  default     = "fourdat-media"
}

# ── App-specific env vars ─────────────────────────────────────────────────────
variable "customer_app_env" {
  description = "Extra env vars for the customer app"
  type        = map(string)
  default     = {}
  sensitive   = true
}

variable "driver_app_env" {
  description = "Extra env vars for the driver app"
  type        = map(string)
  default     = {}
  sensitive   = true
}

variable "merchant_app_env" {
  description = "Extra env vars for the merchant app"
  type        = map(string)
  default     = {}
  sensitive   = true
}

variable "admin_app_env" {
  description = "Extra env vars for the admin app"
  type        = map(string)
  default     = {}
  sensitive   = true
}
