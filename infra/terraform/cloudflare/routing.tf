# ── User-type routing rules ───────────────────────────────────────────────────
#
# Enforces that each user type only lands on their correct subdomain.
# Rules run at the Cloudflare edge before the request hits Pages.
#
# Logic:
#   - A driver hitting fourdat.com        → redirect to ride.fourdat.com
#   - A customer hitting ride.fourdat.com → redirect to fourdat.com
#   - Any unauthenticated hit to admin.*  → pass through (Pages handles auth)
#
# Detection mechanism: we use a __role cookie set by the API on login.
# The cookie value is one of: customer | driver | merchant | admin
#
# Flow:
#   1. User logs in via the API → API sets __role=<type>; Domain=.fourdat.com
#   2. Cloudflare reads the cookie on every subsequent request
#   3. If the role doesn't match the subdomain → 302 to the correct one
# ─────────────────────────────────────────────────────────────────────────────

resource "cloudflare_ruleset" "user_routing" {
  zone_id     = var.cloudflare_zone_id
  name        = "User-type routing"
  description = "Redirect users to their correct app based on __role cookie"
  kind        = "zone"
  phase       = "http_request_dynamic_redirect"

  rules {
    description = "Driver on customer app → ride.fourdat.com"
    enabled     = true
    expression  = "(http.host eq \"fourdat.com\" and http.cookie contains \"__role=driver\")"
    action      = "redirect"
    action_parameters {
      from_value {
        status_code = 302
        target_url {
          expression = "concat(\"https://ride.fourdat.com\", http.request.uri.path)"
        }
        preserve_query_string = true
      }
    }
  }

  rules {
    description = "Merchant on customer app → merchant.fourdat.com"
    enabled     = true
    expression  = "(http.host eq \"fourdat.com\" and http.cookie contains \"__role=merchant\")"
    action      = "redirect"
    action_parameters {
      from_value {
        status_code = 302
        target_url {
          expression = "concat(\"https://merchant.fourdat.com\", http.request.uri.path)"
        }
        preserve_query_string = true
      }
    }
  }

  rules {
    description = "Customer on driver app → fourdat.com"
    enabled     = true
    expression  = "(http.host eq \"ride.fourdat.com\" and http.cookie contains \"__role=customer\")"
    action      = "redirect"
    action_parameters {
      from_value {
        status_code = 302
        target_url {
          expression = "concat(\"https://fourdat.com\", http.request.uri.path)"
        }
        preserve_query_string = true
      }
    }
  }

  rules {
    description = "Merchant on driver app → merchant.fourdat.com"
    enabled     = true
    expression  = "(http.host eq \"ride.fourdat.com\" and http.cookie contains \"__role=merchant\")"
    action      = "redirect"
    action_parameters {
      from_value {
        status_code = 302
        target_url {
          expression = "concat(\"https://merchant.fourdat.com\", http.request.uri.path)"
        }
        preserve_query_string = true
      }
    }
  }

  rules {
    description = "Customer on merchant app → fourdat.com"
    enabled     = true
    expression  = "(http.host eq \"merchant.fourdat.com\" and http.cookie contains \"__role=customer\")"
    action      = "redirect"
    action_parameters {
      from_value {
        status_code = 302
        target_url {
          expression = "concat(\"https://fourdat.com\", http.request.uri.path)"
        }
        preserve_query_string = true
      }
    }
  }

  rules {
    description = "Driver on merchant app → ride.fourdat.com"
    enabled     = true
    expression  = "(http.host eq \"merchant.fourdat.com\" and http.cookie contains \"__role=driver\")"
    action      = "redirect"
    action_parameters {
      from_value {
        status_code = 302
        target_url {
          expression = "concat(\"https://ride.fourdat.com\", http.request.uri.path)"
        }
        preserve_query_string = true
      }
    }
  }

  # admin.fourdat.com — non-admin roles get bounced to their home
  rules {
    description = "Customer on admin app → fourdat.com"
    enabled     = true
    expression  = "(http.host eq \"admin.fourdat.com\" and http.cookie contains \"__role=customer\")"
    action      = "redirect"
    action_parameters {
      from_value {
        status_code = 302
        target_url {
          value = "https://fourdat.com"
        }
        preserve_query_string = false
      }
    }
  }

  rules {
    description = "Driver on admin app → ride.fourdat.com"
    enabled     = true
    expression  = "(http.host eq \"admin.fourdat.com\" and http.cookie contains \"__role=driver\")"
    action      = "redirect"
    action_parameters {
      from_value {
        status_code = 302
        target_url {
          value = "https://ride.fourdat.com"
        }
        preserve_query_string = false
      }
    }
  }

  rules {
    description = "Merchant on admin app → merchant.fourdat.com"
    enabled     = true
    expression  = "(http.host eq \"admin.fourdat.com\" and http.cookie contains \"__role=merchant\")"
    action      = "redirect"
    action_parameters {
      from_value {
        status_code = 302
        target_url {
          value = "https://merchant.fourdat.com"
        }
        preserve_query_string = false
      }
    }
  }
}

# ── Security headers ruleset ──────────────────────────────────────────────────
# Applied to all fourdat.com responses.
resource "cloudflare_ruleset" "security_headers" {
  zone_id     = var.cloudflare_zone_id
  name        = "Security headers"
  description = "Add security headers to all responses"
  kind        = "zone"
  phase       = "http_response_headers_transform"

  rules {
    description = "Add security headers"
    enabled     = true
    expression  = "true"
    action      = "rewrite"
    action_parameters {
      headers {
        name      = "X-Frame-Options"
        operation = "set"
        value     = "SAMEORIGIN"
      }
      headers {
        name      = "X-Content-Type-Options"
        operation = "set"
        value     = "nosniff"
      }
      headers {
        name      = "Referrer-Policy"
        operation = "set"
        value     = "strict-origin-when-cross-origin"
      }
      headers {
        name      = "Permissions-Policy"
        operation = "set"
        value     = "camera=(), microphone=(), geolocation=(self)"
      }
    }
  }
}
