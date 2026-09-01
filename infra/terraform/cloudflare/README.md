# infra/terraform/cloudflare

Manages Cloudflare infrastructure for **fourdat.com**:

- **Pages** — 4 Next.js apps, each on its own subdomain
- **R2** — media bucket with custom domain
- **DNS** — all records for the zone
- **Routing rules** — edge-level user-type isolation via `__role` cookie
- **Security headers** — applied zone-wide

---

## Domain map

| App | Domain | User type |
|-----|--------|-----------|
| customer | `fourdat.com` | customers |
| driver | `ride.fourdat.com` | drivers |
| merchant | `merchant.fourdat.com` | merchants |
| admin | `admin.fourdat.com` | admins |
| API | `api.fourdat.com` | — |
| Media | `media.fourdat.com` | — |

---

## How user-type routing works

On login the Go API sets a cookie:
```
Set-Cookie: __role=<customer|driver|merchant|admin>; Domain=.fourdat.com; Secure; HttpOnly; SameSite=Lax
```

Cloudflare reads this cookie on every request **at the edge** before it reaches Pages. If the role doesn't match the subdomain the user is on, they get a 302 to the correct app:

| Cookie value | Hits | Redirected to |
|---|---|---|
| `driver` | `fourdat.com` | `ride.fourdat.com` |
| `merchant` | `fourdat.com` | `merchant.fourdat.com` |
| `customer` | `ride.fourdat.com` | `fourdat.com` |
| `merchant` | `ride.fourdat.com` | `merchant.fourdat.com` |
| `customer` | `merchant.fourdat.com` | `fourdat.com` |
| `driver` | `merchant.fourdat.com` | `ride.fourdat.com` |
| `customer` | `admin.fourdat.com` | `fourdat.com` |
| `driver` | `admin.fourdat.com` | `ride.fourdat.com` |
| `merchant` | `admin.fourdat.com` | `merchant.fourdat.com` |

No cookie (unauthenticated) → passes through to Pages, which handles the auth redirect internally.

---

## Prerequisites

- [Terraform](https://developer.hashicorp.com/terraform/install) >= 1.6
- Cloudflare account with:
  - `fourdat.com` added as a zone
  - API token with permissions:
    - `Account > Cloudflare Pages > Edit`
    - `Account > R2 Storage > Edit`
    - `Zone > DNS > Edit`
    - `Zone > Transform Rules > Edit`
    - `Zone > Redirect Rules > Edit`
  - R2 bucket `fourdat-tf-state` created manually for Terraform state
  - R2 API token (access key + secret) for the state bucket

---

## First-time setup

### 1. Create the state bucket (once, manually)

Dashboard → R2 → Create bucket → `fourdat-tf-state`

Then: R2 → Manage R2 API Tokens → Create API Token (Object Read & Write).

### 2. Create your tfvars

```bash
cp terraform.tfvars.example terraform.tfvars
# fill in all values
```

### 3. Init with the R2 backend

```bash
terraform init \
  -backend-config="endpoint=https://<ACCOUNT_ID>.r2.cloudflarestorage.com" \
  -backend-config="access_key=<R2_ACCESS_KEY_ID>" \
  -backend-config="secret_key=<R2_SECRET_ACCESS_KEY>"
```

### 4. Plan & apply

```bash
terraform plan
terraform apply
```

---

## After apply — manual steps

### CORS on the media bucket

Dashboard → R2 → `fourdat-media` → Settings → CORS Policy:

```json
[
  {
    "AllowedOrigins": [
      "https://fourdat.com",
      "https://ride.fourdat.com",
      "https://merchant.fourdat.com",
      "https://admin.fourdat.com"
    ],
    "AllowedMethods": ["GET", "PUT", "HEAD"],
    "AllowedHeaders": ["Content-Type", "Content-Length", "Authorization"],
    "ExposeHeaders": ["ETag"],
    "MaxAgeSeconds": 3600
  }
]
```

### Lifecycle rule on the media bucket

Dashboard → R2 → `fourdat-media` → Settings → Object Lifecycle:
- Prefix: `tmp/`
- Expire after: `1 day`

### API sets the __role cookie on login

Make sure your Go API sets this on every successful login response:

```
Set-Cookie: __role=<role>; Domain=.fourdat.com; Path=/; Secure; HttpOnly; SameSite=Lax; Max-Age=86400
```

And clears it on logout:

```
Set-Cookie: __role=; Domain=.fourdat.com; Path=/; Max-Age=0
```

---

## Updating env vars per app

```hcl
# terraform.tfvars
customer_app_env = {
  NEXT_PUBLIC_PAYSTACK_KEY = "pk_live_..."
}
```

Then `terraform apply`.

---

## File structure

```
cloudflare/
├── versions.tf          # provider + R2 backend config
├── variables.tf         # all input variables
├── pages.tf             # Pages projects, custom domains, DNS records
├── r2.tf                # R2 media bucket + custom domain
├── routing.tf           # user-type redirect rules + security headers
├── outputs.tf           # useful URLs after apply
├── terraform.tfvars.example
└── README.md
```
