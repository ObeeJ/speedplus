# SpeedPlus Monorepo Scaffold Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Scaffold a production-ready Turborepo monorepo for SpeedPlus — a multi-vertical essential delivery webapp (gas, grocery, food, pharmacy) targeting underserved African/Nigerian communities.

**Architecture:** Turborepo monorepo with four Next.js 15 apps (customer, driver, merchant, admin) sharing five packages (ui, types, api-client, utils, config). Each app deploys independently. Shared packages enforce type safety and design consistency across all portals.

**Tech Stack:** Bun 1.x · Turborepo 2.x · Next.js 15 · Tailwind v4 · Shadcn UI · Zustand · TanStack Query v5 · React Hook Form · Zod · TypeScript 5.x

## Global Constraints

- Runtime: Bun 1.3.14 — use `bun` for all commands, never `npm` or `yarn`
- Package manager: Bun workspaces — `bun.lockb` is the lockfile
- Next.js: 15 with App Router only — no Pages Router
- Tailwind: v4 — config lives in CSS (`@theme {}`) not `tailwind.config.js`
- State: TanStack Query v5 for server state · Zustand for client/UI state
- Forms: React Hook Form + Zod — no other form libraries
- Brand colors: Primary `#00C48C` (Spearmint) · Secondary `#1A1A2E` (Deep Midnight)
- Typography: Plus Jakarta Sans (display) · Inter (body) · DM Mono (mono)
- No `console.log` in committed code — use proper error boundaries and logging
- All packages use `"type": "module"` and ESM imports
- Path alias `@speedplus/ui`, `@speedplus/types`, `@speedplus/api-client`, `@speedplus/utils` across all apps

---

## File Map

```
/home/obeej/Projects/speedplus/
├── package.json                          # root workspace config
├── turbo.json                            # turborepo pipeline
├── bunfig.toml                           # bun config
├── .gitignore
├── tsconfig.json                         # root TS config (references)
│
├── packages/
│   ├── config/                           # shared tooling config
│   │   ├── package.json
│   │   ├── tailwind/
│   │   │   └── tokens.css               # shared CSS variables & @theme
│   │   ├── eslint/
│   │   │   └── index.js                 # shared ESLint config
│   │   └── typescript/
│   │       ├── base.json                # base tsconfig
│   │       └── nextjs.json              # Next.js tsconfig extends base
│   │
│   ├── types/                           # shared TypeScript types (contract layer)
│   │   ├── package.json
│   │   ├── tsconfig.json
│   │   └── src/
│   │       ├── index.ts                 # barrel export
│   │       ├── common.ts                # ApiResponse, Pagination, ErrorCode
│   │       ├── users.ts                 # User, CustomerProfile, DriverProfile, MerchantProfile
│   │       ├── orders.ts                # Order, OrderItem, OrderStatus, OrderType
│   │       ├── products.ts              # Product, GasProduct, GroceryProduct, FoodItem, PharmacyProduct
│   │       ├── delivery.ts              # Delivery, DeliveryStatus, Location, Route
│   │       └── prescriptions.ts        # Prescription, RxStatus, PrescriptionItem
│   │
│   ├── ui/                              # shared design system (Shadcn + SpeedPlus brand)
│   │   ├── package.json
│   │   ├── tsconfig.json
│   │   ├── components.json              # Shadcn config
│   │   └── src/
│   │       ├── index.ts                 # barrel export
│   │       ├── logo.tsx                 # SpeedPlus SVG logo component
│   │       ├── components/
│   │       │   ├── button.tsx           # Shadcn Button (customised)
│   │       │   ├── input.tsx            # Shadcn Input
│   │       │   ├── badge.tsx            # Shadcn Badge (vertical tags)
│   │       │   └── card.tsx             # Shadcn Card
│   │       └── lib/
│   │           └── utils.ts             # cn() helper
│   │
│   ├── api-client/                      # typed API layer shared by all apps
│   │   ├── package.json
│   │   ├── tsconfig.json
│   │   └── src/
│   │       ├── index.ts
│   │       ├── client.ts                # axios instance, interceptors, error handling
│   │       ├── errors.ts                # SpeedPlusError, error codes
│   │       └── endpoints/
│   │           ├── auth.ts              # login, logout, refreshToken, register
│   │           ├── orders.ts            # createOrder, getOrder, cancelOrder, trackOrder
│   │           ├── products.ts          # getProducts, searchProducts, getProductById
│   │           └── prescriptions.ts     # uploadRx, getRxStatus, getRxHistory
│   │
│   └── utils/                           # shared pure utility functions
│       ├── package.json
│       ├── tsconfig.json
│       └── src/
│           ├── index.ts
│           ├── format.ts                # formatCurrency, formatDate, formatDistance
│           └── validation.ts            # phone, email, prescription file validators
│
└── apps/
    ├── customer/                         # consumer-facing webapp
    │   ├── package.json
    │   ├── tsconfig.json
    │   ├── next.config.ts
    │   ├── postcss.config.mjs
    │   ├── .env.local
    │   └── app/
    │       ├── globals.css               # @import tailwindcss + @theme tokens
    │       ├── layout.tsx                # root layout, fonts, providers
    │       ├── page.tsx                  # landing / vertical selector
    │       ├── providers.tsx             # QueryClientProvider + Zustand
    │       └── (auth)/
    │           ├── login/page.tsx
    │           └── register/page.tsx
    │   └── lib/
    │       ├── store/
    │       │   ├── auth.store.ts         # Zustand: user, token, isAuthenticated
    │       │   ├── cart.store.ts         # Zustand: items, add, remove, clear
    │       │   └── ui.store.ts           # Zustand: modals, toasts, loading
    │       └── query.ts                  # TanStack QueryClient config
    │
    ├── driver/                           # driver portal
    │   ├── package.json
    │   ├── tsconfig.json
    │   ├── next.config.ts
    │   ├── postcss.config.mjs
    │   ├── .env.local
    │   └── app/
    │       ├── globals.css
    │       ├── layout.tsx
    │       ├── page.tsx                  # go online / offline dashboard
    │       └── providers.tsx
    │
    ├── merchant/                         # merchant portal (all 4 verticals)
    │   ├── package.json
    │   ├── tsconfig.json
    │   ├── next.config.ts
    │   ├── postcss.config.mjs
    │   ├── .env.local
    │   └── app/
    │       ├── globals.css
    │       ├── layout.tsx
    │       ├── page.tsx                  # vertical selector dashboard
    │       └── providers.tsx
    │
    └── admin/                            # ops dashboard
        ├── package.json
        ├── tsconfig.json
        ├── next.config.ts
        ├── postcss.config.mjs
        ├── .env.local
        └── app/
            ├── globals.css
            ├── layout.tsx
            ├── page.tsx                  # metrics overview
            └── providers.tsx
```

---

## Task 1: Root Monorepo Configuration

**Files:**
- Create: `package.json`
- Create: `turbo.json`
- Create: `bunfig.toml`
- Create: `.gitignore`
- Create: `tsconfig.json`

**Interfaces:**
- Produces: Bun workspace resolution for all `apps/*` and `packages/*`

- [ ] **Step 1: Create root package.json**

```json
{
  "name": "speedplus",
  "private": true,
  "version": "0.0.1",
  "workspaces": [
    "apps/*",
    "packages/*"
  ],
  "scripts": {
    "build": "turbo build",
    "dev": "turbo dev",
    "dev:customer": "turbo dev --filter=@speedplus/customer",
    "dev:driver": "turbo dev --filter=@speedplus/driver",
    "dev:merchant": "turbo dev --filter=@speedplus/merchant",
    "dev:admin": "turbo dev --filter=@speedplus/admin",
    "lint": "turbo lint",
    "typecheck": "turbo typecheck",
    "clean": "turbo clean && rm -rf node_modules"
  },
  "devDependencies": {
    "turbo": "^2.0.0",
    "typescript": "^5.5.0"
  }
}
```

- [ ] **Step 2: Create turbo.json**

```json
{
  "$schema": "https://turbo.build/schema.json",
  "ui": "tui",
  "tasks": {
    "build": {
      "dependsOn": ["^build"],
      "inputs": ["$TURBO_DEFAULT$", ".env.local"],
      "outputs": [".next/**", "!.next/cache/**", "dist/**"]
    },
    "dev": {
      "cache": false,
      "persistent": true
    },
    "lint": {
      "dependsOn": ["^lint"]
    },
    "typecheck": {
      "dependsOn": ["^typecheck"]
    },
    "clean": {
      "cache": false
    }
  }
}
```

- [ ] **Step 3: Create bunfig.toml**

```toml
[install]
exact = false

[install.lockfile]
save = true

[run]
bun = true
```

- [ ] **Step 4: Create .gitignore**

```
# Dependencies
node_modules
.pnp
.pnp.js

# Build outputs
.next
dist
out
build
.turbo

# Environment
.env
.env.local
.env.production
.env*.local

# Misc
.DS_Store
*.pem
.vercel
*.log
bun.lockb
```

- [ ] **Step 5: Create root tsconfig.json**

```json
{
  "compilerOptions": {
    "strict": true,
    "skipLibCheck": true
  },
  "files": [],
  "references": [
    { "path": "packages/types" },
    { "path": "packages/ui" },
    { "path": "packages/api-client" },
    { "path": "packages/utils" },
    { "path": "apps/customer" },
    { "path": "apps/driver" },
    { "path": "apps/merchant" },
    { "path": "apps/admin" }
  ]
}
```

- [ ] **Step 6: Create all workspace directories**

```bash
mkdir -p packages/{config/{tailwind,eslint,typescript},types/src,ui/src/{components,lib},api-client/src/endpoints,utils/src}
mkdir -p apps/{customer/{app/{(auth)/{login,register}},lib/{store}},driver/app,merchant/app,admin/app}
```

- [ ] **Step 7: Commit**

```bash
cd /home/obeej/Projects/speedplus
git init
git add package.json turbo.json bunfig.toml .gitignore tsconfig.json
git commit -m "chore: initialise speedplus monorepo root"
```

---

## Task 2: Shared Config Package

**Files:**
- Create: `packages/config/package.json`
- Create: `packages/config/tailwind/tokens.css`
- Create: `packages/config/eslint/index.js`
- Create: `packages/config/typescript/base.json`
- Create: `packages/config/typescript/nextjs.json`

**Interfaces:**
- Produces: `@speedplus/config/tailwind/tokens.css` · `@speedplus/config/eslint` · `@speedplus/config/typescript/base.json` · `@speedplus/config/typescript/nextjs.json`

- [ ] **Step 1: Create packages/config/package.json**

```json
{
  "name": "@speedplus/config",
  "version": "0.0.1",
  "private": true,
  "exports": {
    "./tailwind/tokens.css": "./tailwind/tokens.css",
    "./eslint": "./eslint/index.js",
    "./typescript/base.json": "./typescript/base.json",
    "./typescript/nextjs.json": "./typescript/nextjs.json"
  }
}
```

- [ ] **Step 2: Create Tailwind v4 CSS tokens**

```css
/* packages/config/tailwind/tokens.css */

/* Brand Colors */
@theme {
  --color-primary:     #00C48C;
  --color-primary-700: #008F66;
  --color-primary-300: #80E2C6;
  --color-primary-100: #E0FBF4;
  --color-midnight:    #1A1A2E;
  --color-surface:     #F8FFFE;
  --color-mid:         #6B7280;
  --color-error:       #DC2626;
  --color-warning:     #D97706;
  --color-white:       #FFFFFF;

  /* Typography */
  --font-display: 'Plus Jakarta Sans', sans-serif;
  --font-body:    'Inter', sans-serif;
  --font-mono:    'DM Mono', monospace;

  /* Radius */
  --radius-sm:  6px;
  --radius-md:  12px;
  --radius-lg:  20px;
  --radius-xl:  28px;
  --radius-full: 9999px;

  /* Spacing scale (extends Tailwind defaults) */
  --spacing-18: 4.5rem;
  --spacing-22: 5.5rem;
}
```

- [ ] **Step 3: Create shared ESLint config**

```js
// packages/config/eslint/index.js
/** @type {import("eslint").Linter.Config} */
module.exports = {
  extends: [
    "next/core-web-vitals",
    "next/typescript",
  ],
  rules: {
    "no-console": "error",
    "@typescript-eslint/no-unused-vars": ["error", { argsIgnorePattern: "^_" }],
    "@typescript-eslint/no-explicit-any": "error",
  },
};
```

- [ ] **Step 4: Create base TypeScript config**

```json
// packages/config/typescript/base.json
{
  "$schema": "https://json.schemastore.org/tsconfig",
  "compilerOptions": {
    "target": "ES2022",
    "lib": ["ES2022"],
    "module": "ESNext",
    "moduleResolution": "Bundler",
    "strict": true,
    "noUncheckedIndexedAccess": true,
    "exactOptionalPropertyTypes": true,
    "skipLibCheck": true,
    "isolatedModules": true,
    "esModuleInterop": true,
    "resolveJsonModule": true,
    "declaration": true,
    "declarationMap": true,
    "composite": true
  }
}
```

- [ ] **Step 5: Create Next.js TypeScript config**

```json
// packages/config/typescript/nextjs.json
{
  "$schema": "https://json.schemastore.org/tsconfig",
  "extends": "./base.json",
  "compilerOptions": {
    "lib": ["ES2022", "DOM", "DOM.Iterable"],
    "jsx": "preserve",
    "plugins": [{ "name": "next" }],
    "paths": {
      "@/*": ["./src/*", "./*"]
    }
  },
  "exclude": ["node_modules"]
}
```

- [ ] **Step 6: Commit**

```bash
git add packages/config
git commit -m "chore: add shared config package (tailwind tokens, eslint, typescript)"
```

---

## Task 3: Shared Types Package

**Files:**
- Create: `packages/types/package.json`
- Create: `packages/types/tsconfig.json`
- Create: `packages/types/src/common.ts`
- Create: `packages/types/src/users.ts`
- Create: `packages/types/src/orders.ts`
- Create: `packages/types/src/products.ts`
- Create: `packages/types/src/delivery.ts`
- Create: `packages/types/src/prescriptions.ts`
- Create: `packages/types/src/index.ts`

**Interfaces:**
- Produces: All domain types consumed by `api-client`, `ui`, and all apps

- [ ] **Step 1: Create packages/types/package.json**

```json
{
  "name": "@speedplus/types",
  "version": "0.0.1",
  "private": true,
  "main": "./src/index.ts",
  "types": "./src/index.ts",
  "exports": {
    ".": "./src/index.ts"
  }
}
```

- [ ] **Step 2: Create packages/types/tsconfig.json**

```json
{
  "extends": "@speedplus/config/typescript/base.json",
  "compilerOptions": {
    "outDir": "dist",
    "rootDir": "src"
  },
  "include": ["src"],
  "exclude": ["node_modules", "dist"]
}
```

- [ ] **Step 3: Create src/common.ts**

```typescript
// packages/types/src/common.ts

export type ApiResponse<T> = {
  success: true;
  data: T;
  meta?: PaginationMeta;
} | {
  success: false;
  error: ApiError;
};

export interface PaginationMeta {
  page: number;
  pageSize: number;
  total: number;
  totalPages: number;
}

export interface ApiError {
  code: ErrorCode;
  message: string;
  field?: string;
}

export type ErrorCode =
  | 'UNAUTHORIZED'
  | 'FORBIDDEN'
  | 'NOT_FOUND'
  | 'VALIDATION_ERROR'
  | 'RATE_LIMITED'
  | 'INTERNAL_ERROR'
  | 'PRESCRIPTION_REQUIRED'
  | 'OUT_OF_STOCK'
  | 'AREA_NOT_COVERED'
  | 'MERCHANT_CLOSED';

export type Vertical = 'gas' | 'grocery' | 'food' | 'pharmacy';

export type Currency = 'NGN';

export interface Money {
  amount: number;
  currency: Currency;
}

export interface Address {
  id: string;
  label?: string;
  street: string;
  city: string;
  state: string;
  country: string;
  coordinates: Coordinates;
  deliveryInstructions?: string;
}

export interface Coordinates {
  lat: number;
  lng: number;
}

export interface TimeSlot {
  id: string;
  startTime: string;
  endTime: string;
  available: boolean;
}
```

- [ ] **Step 4: Create src/users.ts**

```typescript
// packages/types/src/users.ts

export type UserRole = 'customer' | 'driver' | 'merchant' | 'admin';

export interface User {
  id: string;
  role: UserRole;
  firstName: string;
  lastName: string;
  phone: string;
  email?: string;
  avatarUrl?: string;
  createdAt: string;
  isVerified: boolean;
}

export interface CustomerProfile extends User {
  role: 'customer';
  savedAddresses: import('./common').Address[];
  isFlexPlus: boolean;
}

export type DriverStatus = 'pending' | 'under_review' | 'approved' | 'suspended';
export type VehicleType = 'bicycle' | 'motorcycle' | 'car' | 'van';

export interface DriverProfile extends User {
  role: 'driver';
  status: DriverStatus;
  vehicleType: VehicleType;
  vehiclePlate: string;
  rating: number;
  totalDeliveries: number;
  isOnline: boolean;
}

export type MerchantVertical = 'gas' | 'grocery' | 'food' | 'pharmacy';
export type MerchantStatus = 'pending' | 'active' | 'suspended';

export interface MerchantProfile extends User {
  role: 'merchant';
  businessName: string;
  vertical: MerchantVertical;
  status: MerchantStatus;
  rating: number;
  isOpen: boolean;
  openingHours: OpeningHours;
  licenceNumber?: string;
}

export interface OpeningHours {
  [day: string]: { open: string; close: string } | null;
}
```

- [ ] **Step 5: Create src/orders.ts**

```typescript
// packages/types/src/orders.ts
import type { Address, Money, Vertical } from './common';

export type OrderStatus =
  | 'pending'
  | 'confirmed'
  | 'preparing'
  | 'ready_for_pickup'
  | 'driver_assigned'
  | 'in_transit'
  | 'delivered'
  | 'cancelled'
  | 'refunded';

export interface Order {
  id: string;
  customerId: string;
  merchantId: string;
  driverId?: string;
  vertical: Vertical;
  status: OrderStatus;
  items: OrderItem[];
  deliveryAddress: Address;
  subtotal: Money;
  deliveryFee: Money;
  serviceFee: Money;
  total: Money;
  tip?: Money;
  scheduledFor?: string;
  createdAt: string;
  updatedAt: string;
  estimatedDeliveryAt?: string;
  deliveredAt?: string;
  cancellationReason?: string;
  proofOfDeliveryUrl?: string;
}

export interface OrderItem {
  id: string;
  productId: string;
  name: string;
  quantity: number;
  unitPrice: Money;
  total: Money;
  customizations?: string;
  substitutionPreference?: 'allow' | 'deny' | 'contact';
}

export interface CreateOrderPayload {
  merchantId: string;
  vertical: Vertical;
  items: Array<{ productId: string; quantity: number; customizations?: string }>;
  deliveryAddressId: string;
  tip?: number;
  scheduledFor?: string;
  prescriptionId?: string;
}
```

- [ ] **Step 6: Create src/products.ts**

```typescript
// packages/types/src/products.ts
import type { Money, Vertical } from './common';

export interface BaseProduct {
  id: string;
  merchantId: string;
  vertical: Vertical;
  name: string;
  description?: string;
  price: Money;
  imageUrl?: string;
  isAvailable: boolean;
  category: string;
}

export type CylinderSize = '3kg' | '5kg' | '6kg' | '12.5kg' | '25kg' | '50kg';

export interface GasProduct extends BaseProduct {
  vertical: 'gas';
  cylinderSize: CylinderSize;
  isRefill: boolean;
  requiresEmptyReturn: boolean;
}

export interface GroceryProduct extends BaseProduct {
  vertical: 'grocery';
  unit?: string;
  isWeighted: boolean;
  isAgeRestricted: boolean;
  allergens?: string[];
}

export interface FoodItem extends BaseProduct {
  vertical: 'food';
  modifierGroups?: ModifierGroup[];
  prepTimeMinutes: number;
  isVegetarian: boolean;
  isHalal: boolean;
  calories?: number;
}

export interface ModifierGroup {
  id: string;
  name: string;
  required: boolean;
  minSelections: number;
  maxSelections: number;
  options: ModifierOption[];
}

export interface ModifierOption {
  id: string;
  name: string;
  price: Money;
}

export type RxRequirement = 'none' | 'otc' | 'prescription';

export interface PharmacyProduct extends BaseProduct {
  vertical: 'pharmacy';
  rxRequirement: RxRequirement;
  genericName?: string;
  dosage?: string;
  requiresColdChain: boolean;
}

export type Product = GasProduct | GroceryProduct | FoodItem | PharmacyProduct;
```

- [ ] **Step 7: Create src/delivery.ts**

```typescript
// packages/types/src/delivery.ts
import type { Coordinates } from './common';

export type DeliveryStatus =
  | 'searching_driver'
  | 'driver_assigned'
  | 'heading_to_merchant'
  | 'at_merchant'
  | 'heading_to_customer'
  | 'at_customer'
  | 'delivered'
  | 'failed';

export interface Delivery {
  id: string;
  orderId: string;
  driverId: string;
  status: DeliveryStatus;
  currentLocation?: Coordinates;
  estimatedArrivalMinutes?: number;
  route?: Route;
  proofPhotoUrl?: string;
  failureReason?: string;
}

export interface Route {
  polyline: string;
  distanceKm: number;
  durationMinutes: number;
  waypoints: Coordinates[];
}

export interface DriverLocation {
  driverId: string;
  coordinates: Coordinates;
  heading?: number;
  updatedAt: string;
}
```

- [ ] **Step 8: Create src/prescriptions.ts**

```typescript
// packages/types/src/prescriptions.ts

export type RxStatus =
  | 'uploaded'
  | 'under_review'
  | 'approved'
  | 'rejected'
  | 'expired';

export interface Prescription {
  id: string;
  customerId: string;
  pharmacyId: string;
  status: RxStatus;
  imageUrl: string;
  uploadedAt: string;
  reviewedAt?: string;
  reviewedBy?: string;
  rejectionReason?: string;
  expiresAt?: string;
  items?: PrescriptionItem[];
}

export interface PrescriptionItem {
  productId: string;
  productName: string;
  dosage: string;
  quantity: number;
  refillsRemaining?: number;
}
```

- [ ] **Step 9: Create src/index.ts**

```typescript
// packages/types/src/index.ts
export * from './common';
export * from './users';
export * from './orders';
export * from './products';
export * from './delivery';
export * from './prescriptions';
```

- [ ] **Step 10: Typecheck**

```bash
cd /home/obeej/Projects/speedplus
bun install
cd packages/types && bunx tsc --noEmit
```

Expected: No errors.

- [ ] **Step 11: Commit**

```bash
git add packages/types
git commit -m "feat: add shared types package (all domain contracts)"
```

---

## Task 4: API Client Package

**Files:**
- Create: `packages/api-client/package.json`
- Create: `packages/api-client/tsconfig.json`
- Create: `packages/api-client/src/errors.ts`
- Create: `packages/api-client/src/client.ts`
- Create: `packages/api-client/src/endpoints/auth.ts`
- Create: `packages/api-client/src/endpoints/orders.ts`
- Create: `packages/api-client/src/endpoints/products.ts`
- Create: `packages/api-client/src/endpoints/prescriptions.ts`
- Create: `packages/api-client/src/index.ts`

**Interfaces:**
- Consumes: `@speedplus/types` — all domain types
- Produces: `apiClient`, `authApi`, `ordersApi`, `productsApi`, `prescriptionsApi`

- [ ] **Step 1: Create packages/api-client/package.json**

```json
{
  "name": "@speedplus/api-client",
  "version": "0.0.1",
  "private": true,
  "main": "./src/index.ts",
  "types": "./src/index.ts",
  "exports": {
    ".": "./src/index.ts"
  },
  "dependencies": {
    "@speedplus/types": "workspace:*",
    "axios": "^1.7.0"
  }
}
```

- [ ] **Step 2: Create packages/api-client/tsconfig.json**

```json
{
  "extends": "@speedplus/config/typescript/base.json",
  "compilerOptions": {
    "outDir": "dist",
    "rootDir": "src"
  },
  "include": ["src"],
  "exclude": ["node_modules", "dist"]
}
```

- [ ] **Step 3: Create src/errors.ts**

```typescript
// packages/api-client/src/errors.ts
import type { ErrorCode } from '@speedplus/types';

export class SpeedPlusError extends Error {
  constructor(
    public readonly code: ErrorCode,
    message: string,
    public readonly field?: string,
    public readonly statusCode?: number,
  ) {
    super(message);
    this.name = 'SpeedPlusError';
  }

  static fromAxios(error: unknown): SpeedPlusError {
    if (isAxiosError(error)) {
      const data = error.response?.data;
      if (data?.error) {
        return new SpeedPlusError(
          data.error.code ?? 'INTERNAL_ERROR',
          data.error.message ?? 'An unexpected error occurred',
          data.error.field,
          error.response?.status,
        );
      }
      if (error.response?.status === 401) {
        return new SpeedPlusError('UNAUTHORIZED', 'Session expired. Please log in again.', undefined, 401);
      }
      if (error.response?.status === 403) {
        return new SpeedPlusError('FORBIDDEN', 'You do not have permission to do this.', undefined, 403);
      }
    }
    return new SpeedPlusError('INTERNAL_ERROR', 'An unexpected error occurred');
  }
}

function isAxiosError(error: unknown): error is { response?: { data?: unknown; status?: number }; message: string } {
  return typeof error === 'object' && error !== null && 'response' in error;
}
```

- [ ] **Step 4: Create src/client.ts**

```typescript
// packages/api-client/src/client.ts
import axios, { type AxiosInstance } from 'axios';
import { SpeedPlusError } from './errors';

let authToken: string | null = null;

export function setAuthToken(token: string | null): void {
  authToken = token;
}

export function createApiClient(baseURL: string): AxiosInstance {
  const client = axios.create({
    baseURL,
    timeout: 15_000,
    headers: { 'Content-Type': 'application/json' },
  });

  client.interceptors.request.use((config) => {
    if (authToken) {
      config.headers.Authorization = `Bearer ${authToken}`;
    }
    return config;
  });

  client.interceptors.response.use(
    (response) => response,
    async (error) => {
      const original = error.config;
      if (error.response?.status === 401 && !original._retry) {
        original._retry = true;
        try {
          const { data } = await axios.post(`${baseURL}/auth/refresh`, {}, {
            headers: original.headers,
          });
          setAuthToken(data.data.accessToken);
          original.headers.Authorization = `Bearer ${data.data.accessToken}`;
          return client(original);
        } catch {
          setAuthToken(null);
        }
      }
      throw SpeedPlusError.fromAxios(error);
    },
  );

  return client;
}

export const apiClient = createApiClient(
  process.env.NEXT_PUBLIC_API_URL ?? 'http://localhost:8000/api/v1',
);
```

- [ ] **Step 5: Create src/endpoints/auth.ts**

```typescript
// packages/api-client/src/endpoints/auth.ts
import type { ApiResponse, User } from '@speedplus/types';
import { apiClient, setAuthToken } from '../client';

interface LoginPayload { phone: string; password: string }
interface RegisterPayload { firstName: string; lastName: string; phone: string; password: string }
interface AuthTokens { accessToken: string; refreshToken: string; user: User }

export const authApi = {
  async login(payload: LoginPayload): Promise<AuthTokens> {
    const { data } = await apiClient.post<ApiResponse<AuthTokens>>('/auth/login', payload);
    if (!data.success) throw new Error(data.error.message);
    setAuthToken(data.data.accessToken);
    return data.data;
  },

  async register(payload: RegisterPayload): Promise<AuthTokens> {
    const { data } = await apiClient.post<ApiResponse<AuthTokens>>('/auth/register', payload);
    if (!data.success) throw new Error(data.error.message);
    setAuthToken(data.data.accessToken);
    return data.data;
  },

  async logout(): Promise<void> {
    await apiClient.post('/auth/logout');
    setAuthToken(null);
  },

  async verifyOtp(phone: string, otp: string): Promise<{ verified: boolean }> {
    const { data } = await apiClient.post<ApiResponse<{ verified: boolean }>>('/auth/verify-otp', { phone, otp });
    if (!data.success) throw new Error(data.error.message);
    return data.data;
  },
};
```

- [ ] **Step 6: Create src/endpoints/orders.ts**

```typescript
// packages/api-client/src/endpoints/orders.ts
import type { ApiResponse, Order, CreateOrderPayload, PaginationMeta } from '@speedplus/types';
import { apiClient } from '../client';

export const ordersApi = {
  async create(payload: CreateOrderPayload): Promise<Order> {
    const { data } = await apiClient.post<ApiResponse<Order>>('/orders', payload);
    if (!data.success) throw new Error(data.error.message);
    return data.data;
  },

  async getById(orderId: string): Promise<Order> {
    const { data } = await apiClient.get<ApiResponse<Order>>(`/orders/${orderId}`);
    if (!data.success) throw new Error(data.error.message);
    return data.data;
  },

  async list(params?: { page?: number; status?: string }): Promise<{ orders: Order[]; meta: PaginationMeta }> {
    const { data } = await apiClient.get<ApiResponse<{ orders: Order[]; meta: PaginationMeta }>>('/orders', { params });
    if (!data.success) throw new Error(data.error.message);
    return data.data;
  },

  async cancel(orderId: string, reason: string): Promise<Order> {
    const { data } = await apiClient.post<ApiResponse<Order>>(`/orders/${orderId}/cancel`, { reason });
    if (!data.success) throw new Error(data.error.message);
    return data.data;
  },

  async track(orderId: string): Promise<{ order: Order }> {
    const { data } = await apiClient.get<ApiResponse<{ order: Order }>>(`/orders/${orderId}/track`);
    if (!data.success) throw new Error(data.error.message);
    return data.data;
  },
};
```

- [ ] **Step 7: Create src/endpoints/products.ts**

```typescript
// packages/api-client/src/endpoints/products.ts
import type { ApiResponse, Product, Vertical, PaginationMeta } from '@speedplus/types';
import { apiClient } from '../client';

export const productsApi = {
  async list(params: {
    vertical: Vertical;
    merchantId?: string;
    category?: string;
    page?: number;
  }): Promise<{ products: Product[]; meta: PaginationMeta }> {
    const { data } = await apiClient.get<ApiResponse<{ products: Product[]; meta: PaginationMeta }>>('/products', { params });
    if (!data.success) throw new Error(data.error.message);
    return data.data;
  },

  async getById(productId: string): Promise<Product> {
    const { data } = await apiClient.get<ApiResponse<Product>>(`/products/${productId}`);
    if (!data.success) throw new Error(data.error.message);
    return data.data;
  },

  async search(query: string, vertical?: Vertical): Promise<Product[]> {
    const { data } = await apiClient.get<ApiResponse<Product[]>>('/products/search', {
      params: { q: query, vertical },
    });
    if (!data.success) throw new Error(data.error.message);
    return data.data;
  },
};
```

- [ ] **Step 8: Create src/endpoints/prescriptions.ts**

```typescript
// packages/api-client/src/endpoints/prescriptions.ts
import type { ApiResponse, Prescription } from '@speedplus/types';
import { apiClient } from '../client';

export const prescriptionsApi = {
  async upload(pharmacyId: string, imageFile: File): Promise<Prescription> {
    const form = new FormData();
    form.append('pharmacyId', pharmacyId);
    form.append('image', imageFile);
    const { data } = await apiClient.post<ApiResponse<Prescription>>('/prescriptions', form, {
      headers: { 'Content-Type': 'multipart/form-data' },
    });
    if (!data.success) throw new Error(data.error.message);
    return data.data;
  },

  async getById(prescriptionId: string): Promise<Prescription> {
    const { data } = await apiClient.get<ApiResponse<Prescription>>(`/prescriptions/${prescriptionId}`);
    if (!data.success) throw new Error(data.error.message);
    return data.data;
  },

  async list(): Promise<Prescription[]> {
    const { data } = await apiClient.get<ApiResponse<Prescription[]>>('/prescriptions');
    if (!data.success) throw new Error(data.error.message);
    return data.data;
  },
};
```

- [ ] **Step 9: Create src/index.ts**

```typescript
// packages/api-client/src/index.ts
export { apiClient, setAuthToken } from './client';
export { SpeedPlusError } from './errors';
export { authApi } from './endpoints/auth';
export { ordersApi } from './endpoints/orders';
export { productsApi } from './endpoints/products';
export { prescriptionsApi } from './endpoints/prescriptions';
```

- [ ] **Step 10: Commit**

```bash
git add packages/api-client
git commit -m "feat: add typed api-client package with axios, interceptors and all endpoints"
```

---

## Task 5: Utils Package

**Files:**
- Create: `packages/utils/package.json`
- Create: `packages/utils/tsconfig.json`
- Create: `packages/utils/src/format.ts`
- Create: `packages/utils/src/validation.ts`
- Create: `packages/utils/src/index.ts`

**Interfaces:**
- Produces: `formatCurrency`, `formatDate`, `formatDistance`, `isValidPhone`, `isValidEmail`, `isValidPrescriptionFile`

- [ ] **Step 1: Create packages/utils/package.json**

```json
{
  "name": "@speedplus/utils",
  "version": "0.0.1",
  "private": true,
  "main": "./src/index.ts",
  "types": "./src/index.ts",
  "exports": {
    ".": "./src/index.ts"
  },
  "dependencies": {
    "@speedplus/types": "workspace:*"
  }
}
```

- [ ] **Step 2: Create packages/utils/tsconfig.json**

```json
{
  "extends": "@speedplus/config/typescript/base.json",
  "compilerOptions": { "outDir": "dist", "rootDir": "src" },
  "include": ["src"],
  "exclude": ["node_modules", "dist"]
}
```

- [ ] **Step 3: Create src/format.ts**

```typescript
// packages/utils/src/format.ts
import type { Money } from '@speedplus/types';

export function formatCurrency(money: Money): string {
  return new Intl.NumberFormat('en-NG', {
    style: 'currency',
    currency: money.currency,
    minimumFractionDigits: 0,
    maximumFractionDigits: 2,
  }).format(money.amount / 100);
}

export function formatDate(iso: string): string {
  return new Intl.DateTimeFormat('en-NG', {
    day: 'numeric',
    month: 'short',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  }).format(new Date(iso));
}

export function formatDistance(km: number): string {
  if (km < 1) return `${Math.round(km * 1000)}m away`;
  return `${km.toFixed(1)}km away`;
}

export function formatEta(minutes: number): string {
  if (minutes < 60) return `${minutes} min`;
  const h = Math.floor(minutes / 60);
  const m = minutes % 60;
  return m > 0 ? `${h}h ${m}min` : `${h}h`;
}
```

- [ ] **Step 4: Create src/validation.ts**

```typescript
// packages/utils/src/validation.ts

const NIGERIA_PHONE_REGEX = /^(\+234|0)[789][01]\d{8}$/;
const ALLOWED_RX_TYPES = ['image/jpeg', 'image/png', 'image/webp', 'application/pdf'];
const MAX_RX_SIZE_BYTES = 10 * 1024 * 1024; // 10MB

export function isValidPhone(phone: string): boolean {
  return NIGERIA_PHONE_REGEX.test(phone.replace(/\s/g, ''));
}

export function isValidEmail(email: string): boolean {
  return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email);
}

export function isValidPrescriptionFile(file: File): { valid: boolean; error?: string } {
  if (!ALLOWED_RX_TYPES.includes(file.type)) {
    return { valid: false, error: 'File must be JPG, PNG, WebP, or PDF' };
  }
  if (file.size > MAX_RX_SIZE_BYTES) {
    return { valid: false, error: 'File must be smaller than 10MB' };
  }
  return { valid: true };
}
```

- [ ] **Step 5: Create src/index.ts**

```typescript
export * from './format';
export * from './validation';
```

- [ ] **Step 6: Commit**

```bash
git add packages/utils
git commit -m "feat: add utils package (formatCurrency, formatDate, validators)"
```

---

## Task 6: UI Package — Logo + Components

**Files:**
- Create: `packages/ui/package.json`
- Create: `packages/ui/tsconfig.json`
- Create: `packages/ui/components.json`
- Create: `packages/ui/src/lib/utils.ts`
- Create: `packages/ui/src/logo.tsx`
- Create: `packages/ui/src/components/button.tsx`
- Create: `packages/ui/src/components/input.tsx`
- Create: `packages/ui/src/components/badge.tsx`
- Create: `packages/ui/src/components/card.tsx`
- Create: `packages/ui/src/index.ts`

**Interfaces:**
- Produces: `<SpeedPlusLogo />`, `<Button />`, `<Input />`, `<Badge />`, `<Card />`, `cn()`

- [ ] **Step 1: Create packages/ui/package.json**

```json
{
  "name": "@speedplus/ui",
  "version": "0.0.1",
  "private": true,
  "main": "./src/index.ts",
  "types": "./src/index.ts",
  "exports": {
    ".": "./src/index.ts"
  },
  "dependencies": {
    "class-variance-authority": "^0.7.0",
    "clsx": "^2.1.1",
    "tailwind-merge": "^2.5.0",
    "lucide-react": "^0.454.0"
  },
  "peerDependencies": {
    "react": "^19.0.0",
    "react-dom": "^19.0.0"
  }
}
```

- [ ] **Step 2: Create packages/ui/tsconfig.json**

```json
{
  "extends": "@speedplus/config/typescript/nextjs.json",
  "compilerOptions": {
    "outDir": "dist",
    "rootDir": "src"
  },
  "include": ["src"],
  "exclude": ["node_modules", "dist"]
}
```

- [ ] **Step 3: Create packages/ui/components.json (Shadcn config)**

```json
{
  "$schema": "https://ui.shadcn.com/schema.json",
  "style": "default",
  "rsc": true,
  "tsx": true,
  "tailwind": {
    "config": "",
    "css": "src/styles/globals.css",
    "baseColor": "neutral",
    "cssVariables": true
  },
  "aliases": {
    "components": "@speedplus/ui/components",
    "utils": "@speedplus/ui/lib/utils"
  }
}
```

- [ ] **Step 4: Create src/lib/utils.ts**

```typescript
// packages/ui/src/lib/utils.ts
import { clsx, type ClassValue } from 'clsx';
import { twMerge } from 'tailwind-merge';

export function cn(...inputs: ClassValue[]): string {
  return twMerge(clsx(inputs));
}
```

- [ ] **Step 5: Create the SpeedPlus SVG Logo component**

```tsx
// packages/ui/src/logo.tsx
import type { SVGProps } from 'react';

interface SpeedPlusLogoProps {
  variant?: 'full' | 'mark' | 'wordmark';
  theme?: 'dark' | 'light';
  size?: 'sm' | 'md' | 'lg' | 'xl';
  className?: string;
}

const sizes = {
  sm: { markSize: 24, fontSize: 16, gap: 8 },
  md: { markSize: 32, fontSize: 22, gap: 10 },
  lg: { markSize: 44, fontSize: 30, gap: 14 },
  xl: { markSize: 64, fontSize: 44, gap: 20 },
};

function PlusMark({
  size,
  color,
  ...props
}: { size: number; color: string } & SVGProps<SVGSVGElement>) {
  const arm = Math.round(size * 0.2);   // bar thickness
  const len = Math.round(size * 0.62);  // bar length
  const r = Math.round(arm / 2);        // corner radius
  const cx = size / 2;
  const cy = size / 2;

  return (
    <svg
      width={size}
      height={size}
      viewBox={`0 0 ${size} ${size}`}
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      aria-hidden="true"
      {...props}
    >
      {/* Vertical bar */}
      <rect
        x={cx - arm / 2}
        y={cy - len / 2}
        width={arm}
        height={len}
        rx={r}
        fill={color}
      />
      {/* Horizontal bar */}
      <rect
        x={cx - len / 2}
        y={cy - arm / 2}
        width={len}
        height={arm}
        rx={r}
        fill={color}
      />
    </svg>
  );
}

export function SpeedPlusLogo({
  variant = 'full',
  theme = 'dark',
  size = 'md',
  className,
}: SpeedPlusLogoProps) {
  const { markSize, fontSize, gap } = sizes[size];
  const primaryColor = '#00C48C';
  const textColor = theme === 'dark' ? '#FFFFFF' : '#1A1A2E';
  const markColor = variant === 'mark' && theme === 'light' ? '#1A1A2E' : primaryColor;

  if (variant === 'mark') {
    return (
      <PlusMark
        size={markSize}
        color={markColor}
        className={className}
        role="img"
        aria-label="SpeedPlus"
      />
    );
  }

  if (variant === 'wordmark') {
    return (
      <svg
        height={fontSize * 1.2}
        viewBox={`0 0 ${fontSize * 4.8} ${fontSize * 1.2}`}
        fill="none"
        xmlns="http://www.w3.org/2000/svg"
        className={className}
        role="img"
        aria-label="SpeedPlus"
      >
        <text
          y={fontSize}
          fontFamily="'Plus Jakarta Sans', Inter, sans-serif"
          fontWeight="800"
          fontSize={fontSize}
          fill={textColor}
        >
          Speed
          <tspan fill={primaryColor}>+</tspan>
        </text>
      </svg>
    );
  }

  // full — mark + wordmark
  const totalWidth = markSize + gap + fontSize * 4.8;
  const totalHeight = Math.max(markSize, fontSize * 1.2);

  return (
    <svg
      width={totalWidth}
      height={totalHeight}
      viewBox={`0 0 ${totalWidth} ${totalHeight}`}
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      className={className}
      role="img"
      aria-label="SpeedPlus"
    >
      {/* Plus mark */}
      <g transform={`translate(0, ${(totalHeight - markSize) / 2})`}>
        <PlusMark size={markSize} color={primaryColor} />
      </g>
      {/* Wordmark */}
      <text
        x={markSize + gap}
        y={(totalHeight + fontSize * 0.75) / 2}
        fontFamily="'Plus Jakarta Sans', Inter, sans-serif"
        fontWeight="800"
        fontSize={fontSize}
        fill={textColor}
      >
        Speed
        <tspan fill={primaryColor}>+</tspan>
      </text>
    </svg>
  );
}
```

- [ ] **Step 6: Create src/components/button.tsx**

```tsx
// packages/ui/src/components/button.tsx
import { cva, type VariantProps } from 'class-variance-authority';
import { cn } from '../lib/utils';
import { type ButtonHTMLAttributes, forwardRef } from 'react';

const buttonVariants = cva(
  'inline-flex items-center justify-center gap-2 whitespace-nowrap rounded-xl font-semibold transition-all focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-offset-2 disabled:pointer-events-none disabled:opacity-50 active:scale-[0.98]',
  {
    variants: {
      variant: {
        primary:   'bg-primary text-white hover:bg-primary-700 focus-visible:ring-primary',
        secondary: 'bg-midnight text-white hover:bg-midnight/90 focus-visible:ring-midnight',
        outline:   'border-2 border-primary text-primary hover:bg-primary-100 focus-visible:ring-primary',
        ghost:     'text-midnight hover:bg-primary-100 focus-visible:ring-primary',
        danger:    'bg-error text-white hover:bg-error/90 focus-visible:ring-error',
      },
      size: {
        sm:   'h-9 px-4 text-sm',
        md:   'h-11 px-6 text-base',
        lg:   'h-13 px-8 text-lg',
        icon: 'h-10 w-10',
      },
    },
    defaultVariants: { variant: 'primary', size: 'md' },
  },
);

export interface ButtonProps
  extends ButtonHTMLAttributes<HTMLButtonElement>,
    VariantProps<typeof buttonVariants> {
  isLoading?: boolean;
}

export const Button = forwardRef<HTMLButtonElement, ButtonProps>(
  ({ className, variant, size, isLoading, children, disabled, ...props }, ref) => (
    <button
      ref={ref}
      className={cn(buttonVariants({ variant, size }), className)}
      disabled={disabled || isLoading}
      {...props}
    >
      {isLoading ? (
        <svg className="animate-spin h-4 w-4" viewBox="0 0 24 24" fill="none">
          <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
          <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
        </svg>
      ) : null}
      {children}
    </button>
  ),
);
Button.displayName = 'Button';

export { buttonVariants };
```

- [ ] **Step 7: Create src/components/input.tsx**

```tsx
// packages/ui/src/components/input.tsx
import { forwardRef, type InputHTMLAttributes } from 'react';
import { cn } from '../lib/utils';

export interface InputProps extends InputHTMLAttributes<HTMLInputElement> {
  label?: string;
  error?: string;
  hint?: string;
}

export const Input = forwardRef<HTMLInputElement, InputProps>(
  ({ className, label, error, hint, id, ...props }, ref) => (
    <div className="flex flex-col gap-1.5">
      {label ? (
        <label htmlFor={id} className="text-sm font-medium text-midnight">
          {label}
        </label>
      ) : null}
      <input
        ref={ref}
        id={id}
        className={cn(
          'h-11 w-full rounded-xl border bg-white px-4 text-base text-midnight placeholder:text-mid',
          'focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent',
          'disabled:cursor-not-allowed disabled:opacity-50',
          error ? 'border-error focus:ring-error' : 'border-primary-300',
          className,
        )}
        {...props}
      />
      {error ? <p className="text-xs text-error">{error}</p> : null}
      {hint && !error ? <p className="text-xs text-mid">{hint}</p> : null}
    </div>
  ),
);
Input.displayName = 'Input';
```

- [ ] **Step 8: Create src/components/badge.tsx**

```tsx
// packages/ui/src/components/badge.tsx
import { cva, type VariantProps } from 'class-variance-authority';
import { cn } from '../lib/utils';
import type { HTMLAttributes } from 'react';

const badgeVariants = cva(
  'inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-semibold transition-colors',
  {
    variants: {
      variant: {
        gas:      'bg-primary-100 text-primary-700',
        grocery:  'bg-green-100 text-green-700',
        food:     'bg-amber-100 text-amber-700',
        pharmacy: 'bg-teal-100 text-teal-700',
        default:  'bg-primary-100 text-primary-700',
        success:  'bg-green-100 text-green-700',
        warning:  'bg-amber-100 text-amber-800',
        error:    'bg-red-100 text-red-700',
      },
    },
    defaultVariants: { variant: 'default' },
  },
);

export interface BadgeProps
  extends HTMLAttributes<HTMLSpanElement>,
    VariantProps<typeof badgeVariants> {}

export function Badge({ className, variant, ...props }: BadgeProps) {
  return <span className={cn(badgeVariants({ variant }), className)} {...props} />;
}
```

- [ ] **Step 9: Create src/components/card.tsx**

```tsx
// packages/ui/src/components/card.tsx
import { cn } from '../lib/utils';
import type { HTMLAttributes } from 'react';

export function Card({ className, ...props }: HTMLAttributes<HTMLDivElement>) {
  return (
    <div
      className={cn('rounded-2xl bg-white shadow-sm border border-primary-100 p-4', className)}
      {...props}
    />
  );
}

export function CardHeader({ className, ...props }: HTMLAttributes<HTMLDivElement>) {
  return <div className={cn('mb-3', className)} {...props} />;
}

export function CardTitle({ className, ...props }: HTMLAttributes<HTMLHeadingElement>) {
  return <h3 className={cn('text-base font-semibold text-midnight', className)} {...props} />;
}

export function CardContent({ className, ...props }: HTMLAttributes<HTMLDivElement>) {
  return <div className={cn('text-sm text-mid', className)} {...props} />;
}
```

- [ ] **Step 10: Create src/index.ts**

```typescript
// packages/ui/src/index.ts
export { SpeedPlusLogo } from './logo';
export { Button, buttonVariants, type ButtonProps } from './components/button';
export { Input, type InputProps } from './components/input';
export { Badge, type BadgeProps } from './components/badge';
export { Card, CardHeader, CardTitle, CardContent } from './components/card';
export { cn } from './lib/utils';
```

- [ ] **Step 11: Install UI package deps**

```bash
cd /home/obeej/Projects/speedplus
bun install
```

- [ ] **Step 12: Commit**

```bash
git add packages/ui
git commit -m "feat: add UI package with SpeedPlus SVG logo, Button, Input, Badge, Card"
```

---

## Task 7: Customer App Scaffold

**Files:**
- Create: `apps/customer/package.json`
- Create: `apps/customer/tsconfig.json`
- Create: `apps/customer/next.config.ts`
- Create: `apps/customer/postcss.config.mjs`
- Create: `apps/customer/.env.local`
- Create: `apps/customer/app/globals.css`
- Create: `apps/customer/app/layout.tsx`
- Create: `apps/customer/app/providers.tsx`
- Create: `apps/customer/app/page.tsx`
- Create: `apps/customer/lib/query.ts`
- Create: `apps/customer/lib/store/auth.store.ts`
- Create: `apps/customer/lib/store/cart.store.ts`
- Create: `apps/customer/lib/store/ui.store.ts`

**Interfaces:**
- Consumes: `@speedplus/ui`, `@speedplus/types`, `@speedplus/api-client`, `@speedplus/utils`
- Produces: Running Next.js 15 app on `localhost:3000`

- [ ] **Step 1: Create apps/customer/package.json**

```json
{
  "name": "@speedplus/customer",
  "version": "0.0.1",
  "private": true,
  "scripts": {
    "dev": "next dev --turbopack --port 3000",
    "build": "next build",
    "start": "next start",
    "lint": "next lint",
    "typecheck": "tsc --noEmit"
  },
  "dependencies": {
    "@speedplus/ui": "workspace:*",
    "@speedplus/types": "workspace:*",
    "@speedplus/api-client": "workspace:*",
    "@speedplus/utils": "workspace:*",
    "@tanstack/react-query": "^5.59.0",
    "@tanstack/react-query-devtools": "^5.59.0",
    "next": "^15.0.0",
    "react": "^19.0.0",
    "react-dom": "^19.0.0",
    "zustand": "^5.0.0",
    "react-hook-form": "^7.53.0",
    "zod": "^3.23.0",
    "@hookform/resolvers": "^3.9.0"
  },
  "devDependencies": {
    "@speedplus/config": "workspace:*",
    "@types/node": "^22.0.0",
    "@types/react": "^19.0.0",
    "@types/react-dom": "^19.0.0",
    "tailwindcss": "^4.0.0",
    "@tailwindcss/postcss": "^4.0.0"
  }
}
```

- [ ] **Step 2: Create apps/customer/tsconfig.json**

```json
{
  "extends": "@speedplus/config/typescript/nextjs.json",
  "compilerOptions": {
    "baseUrl": ".",
    "paths": {
      "@/*": ["./*"],
      "@speedplus/ui": ["../../packages/ui/src/index.ts"],
      "@speedplus/types": ["../../packages/types/src/index.ts"],
      "@speedplus/api-client": ["../../packages/api-client/src/index.ts"],
      "@speedplus/utils": ["../../packages/utils/src/index.ts"]
    }
  },
  "include": ["next-env.d.ts", "**/*.ts", "**/*.tsx", ".next/types/**/*.ts"],
  "exclude": ["node_modules"]
}
```

- [ ] **Step 3: Create apps/customer/next.config.ts**

```typescript
import type { NextConfig } from 'next';

const nextConfig: NextConfig = {
  transpilePackages: [
    '@speedplus/ui',
    '@speedplus/types',
    '@speedplus/api-client',
    '@speedplus/utils',
  ],
  images: {
    remotePatterns: [
      { protocol: 'https', hostname: '**.speedplus.ng' },
    ],
  },
};

export default nextConfig;
```

- [ ] **Step 4: Create apps/customer/postcss.config.mjs**

```js
export default {
  plugins: {
    '@tailwindcss/postcss': {},
  },
};
```

- [ ] **Step 5: Create apps/customer/.env.local**

```env
NEXT_PUBLIC_API_URL=http://localhost:8000/api/v1
NEXT_PUBLIC_GOOGLE_MAPS_KEY=your_key_here
NEXT_PUBLIC_APP_ENV=development
```

- [ ] **Step 6: Create apps/customer/app/globals.css**

```css
@import "tailwindcss";
@import "@speedplus/config/tailwind/tokens.css";

@layer base {
  *, *::before, *::after { box-sizing: border-box; }

  html {
    -webkit-tap-highlight-color: transparent;
    scroll-behavior: smooth;
  }

  body {
    font-family: var(--font-body);
    background-color: var(--color-surface);
    color: var(--color-midnight);
    -webkit-font-smoothing: antialiased;
  }

  h1, h2, h3, h4, h5, h6 {
    font-family: var(--font-display);
    font-weight: 700;
  }
}
```

- [ ] **Step 7: Create apps/customer/lib/query.ts**

```typescript
// apps/customer/lib/query.ts
import { QueryClient } from '@tanstack/react-query';

export function makeQueryClient(): QueryClient {
  return new QueryClient({
    defaultOptions: {
      queries: {
        staleTime: 60 * 1000,         // 1 minute
        gcTime: 5 * 60 * 1000,        // 5 minutes
        retry: (count, error) => {
          if (error instanceof Error && error.message.includes('UNAUTHORIZED')) return false;
          return count < 2;
        },
        refetchOnWindowFocus: false,
      },
      mutations: {
        retry: false,
      },
    },
  });
}

let browserQueryClient: QueryClient | undefined;

export function getQueryClient(): QueryClient {
  if (typeof window === 'undefined') return makeQueryClient();
  browserQueryClient ??= makeQueryClient();
  return browserQueryClient;
}
```

- [ ] **Step 8: Create Zustand auth store**

```typescript
// apps/customer/lib/store/auth.store.ts
import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import type { CustomerProfile } from '@speedplus/types';
import { setAuthToken } from '@speedplus/api-client';

interface AuthState {
  user: CustomerProfile | null;
  accessToken: string | null;
  isAuthenticated: boolean;
  setAuth: (user: CustomerProfile, accessToken: string) => void;
  clearAuth: () => void;
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set) => ({
      user: null,
      accessToken: null,
      isAuthenticated: false,

      setAuth: (user, accessToken) => {
        setAuthToken(accessToken);
        set({ user, accessToken, isAuthenticated: true });
      },

      clearAuth: () => {
        setAuthToken(null);
        set({ user: null, accessToken: null, isAuthenticated: false });
      },
    }),
    {
      name: 'speedplus-auth',
      partialize: (state) => ({
        user: state.user,
        accessToken: state.accessToken,
        isAuthenticated: state.isAuthenticated,
      }),
    },
  ),
);
```

- [ ] **Step 9: Create Zustand cart store**

```typescript
// apps/customer/lib/store/cart.store.ts
import { create } from 'zustand';
import type { Vertical } from '@speedplus/types';

interface CartItem {
  productId: string;
  name: string;
  price: number;
  quantity: number;
  customizations?: string;
  imageUrl?: string;
}

interface CartState {
  merchantId: string | null;
  vertical: Vertical | null;
  items: CartItem[];
  addItem: (merchantId: string, vertical: Vertical, item: CartItem) => void;
  removeItem: (productId: string) => void;
  updateQuantity: (productId: string, quantity: number) => void;
  clearCart: () => void;
  totalItems: () => number;
  subtotal: () => number;
}

export const useCartStore = create<CartState>()((set, get) => ({
  merchantId: null,
  vertical: null,
  items: [],

  addItem: (merchantId, vertical, item) => {
    const current = get();
    if (current.merchantId && current.merchantId !== merchantId) {
      set({ merchantId, vertical, items: [item] });
      return;
    }
    const existing = current.items.find((i) => i.productId === item.productId);
    if (existing) {
      set({
        items: current.items.map((i) =>
          i.productId === item.productId
            ? { ...i, quantity: i.quantity + item.quantity }
            : i,
        ),
      });
    } else {
      set({ merchantId, vertical, items: [...current.items, item] });
    }
  },

  removeItem: (productId) =>
    set((s) => ({ items: s.items.filter((i) => i.productId !== productId) })),

  updateQuantity: (productId, quantity) =>
    set((s) => ({
      items: quantity <= 0
        ? s.items.filter((i) => i.productId !== productId)
        : s.items.map((i) => (i.productId === productId ? { ...i, quantity } : i)),
    })),

  clearCart: () => set({ merchantId: null, vertical: null, items: [] }),

  totalItems: () => get().items.reduce((sum, i) => sum + i.quantity, 0),

  subtotal: () => get().items.reduce((sum, i) => sum + i.price * i.quantity, 0),
}));
```

- [ ] **Step 10: Create Zustand UI store**

```typescript
// apps/customer/lib/store/ui.store.ts
import { create } from 'zustand';

interface Toast {
  id: string;
  message: string;
  type: 'success' | 'error' | 'info';
}

interface UiState {
  toasts: Toast[];
  isCartOpen: boolean;
  addToast: (message: string, type?: Toast['type']) => void;
  removeToast: (id: string) => void;
  setCartOpen: (open: boolean) => void;
}

export const useUiStore = create<UiState>()((set) => ({
  toasts: [],
  isCartOpen: false,

  addToast: (message, type = 'info') => {
    const id = crypto.randomUUID();
    set((s) => ({ toasts: [...s.toasts, { id, message, type }] }));
    setTimeout(() => set((s) => ({ toasts: s.toasts.filter((t) => t.id !== id) })), 4000);
  },

  removeToast: (id) => set((s) => ({ toasts: s.toasts.filter((t) => t.id !== id) })),
  setCartOpen: (open) => set({ isCartOpen: open }),
}));
```

- [ ] **Step 11: Create apps/customer/app/providers.tsx**

```tsx
// apps/customer/app/providers.tsx
'use client';

import { QueryClientProvider } from '@tanstack/react-query';
import { ReactQueryDevtools } from '@tanstack/react-query-devtools';
import { getQueryClient } from '@/lib/query';
import type { ReactNode } from 'react';

export function Providers({ children }: { children: ReactNode }) {
  const queryClient = getQueryClient();
  return (
    <QueryClientProvider client={queryClient}>
      {children}
      <ReactQueryDevtools initialIsOpen={false} />
    </QueryClientProvider>
  );
}
```

- [ ] **Step 12: Create apps/customer/app/layout.tsx**

```tsx
// apps/customer/app/layout.tsx
import type { Metadata } from 'next';
import { Inter } from 'next/font/google';
import { Plus_Jakarta_Sans } from 'next/font/google';
import { Providers } from './providers';
import './globals.css';

const inter = Inter({
  subsets: ['latin'],
  variable: '--font-body',
  display: 'swap',
});

const plusJakartaSans = Plus_Jakarta_Sans({
  subsets: ['latin'],
  variable: '--font-display',
  weight: ['400', '500', '600', '700', '800'],
  display: 'swap',
});

export const metadata: Metadata = {
  title: 'SpeedPlus — Faster. Cheaper. Better.',
  description: 'Gas, groceries, food and pharmacy delivered fast across Nigeria.',
  themeColor: '#00C48C',
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en" className={`${inter.variable} ${plusJakartaSans.variable}`}>
      <body>
        <Providers>{children}</Providers>
      </body>
    </html>
  );
}
```

- [ ] **Step 13: Create apps/customer/app/page.tsx**

```tsx
// apps/customer/app/page.tsx
import { SpeedPlusLogo } from '@speedplus/ui';
import type { Vertical } from '@speedplus/types';

const verticals: Array<{ id: Vertical; label: string; emoji: string; description: string }> = [
  { id: 'gas',      label: 'Cooking Gas',  emoji: '🔥', description: 'Cylinders delivered in 30 min' },
  { id: 'grocery',  label: 'Grocery',      emoji: '🛒', description: 'Fresh produce & essentials' },
  { id: 'food',     label: 'Food',         emoji: '🍽️', description: 'Restaurants & home chefs' },
  { id: 'pharmacy', label: 'Pharmacy',     emoji: '💊', description: 'OTC & prescription meds' },
];

export default function HomePage() {
  return (
    <main className="min-h-screen bg-midnight flex flex-col">
      <header className="px-6 pt-12 pb-8">
        <SpeedPlusLogo variant="full" theme="dark" size="lg" />
        <p className="mt-3 text-primary-300 text-sm font-medium tracking-wide">
          Faster. Cheaper. Better.
        </p>
      </header>

      <section className="px-6 flex-1">
        <h1 className="text-white text-2xl font-bold mb-2">
          What do you need?
        </h1>
        <p className="text-mid text-sm mb-6">
          Choose a category to get started
        </p>

        <div className="grid grid-cols-2 gap-4">
          {verticals.map((v) => (
            <a
              key={v.id}
              href={`/${v.id}`}
              className="group rounded-2xl bg-white/5 border border-white/10 p-5 hover:bg-white/10 hover:border-primary/40 transition-all"
            >
              <div className="text-3xl mb-3">{v.emoji}</div>
              <div className="text-white font-semibold text-base">{v.label}</div>
              <div className="text-mid text-xs mt-1">{v.description}</div>
            </a>
          ))}
        </div>
      </section>
    </main>
  );
}
```

- [ ] **Step 14: Install all dependencies and run dev**

```bash
cd /home/obeej/Projects/speedplus
bun install
bun dev:customer
```

Expected: Next.js starts on `http://localhost:3000`. No TypeScript errors. Page renders with SpeedPlus logo, dark midnight background, and four vertical cards.

- [ ] **Step 15: Commit**

```bash
git add apps/customer
git commit -m "feat: scaffold customer app with Zustand stores, TanStack Query, SpeedPlus homepage"
```

---

## Task 8: Driver, Merchant, Admin App Scaffolds

**Files:** Same pattern as customer for each app.

- [ ] **Step 1: Create apps/driver/package.json**

```json
{
  "name": "@speedplus/driver",
  "version": "0.0.1",
  "private": true,
  "scripts": {
    "dev": "next dev --turbopack --port 3001",
    "build": "next build",
    "start": "next start",
    "typecheck": "tsc --noEmit"
  },
  "dependencies": {
    "@speedplus/ui": "workspace:*",
    "@speedplus/types": "workspace:*",
    "@speedplus/api-client": "workspace:*",
    "@speedplus/utils": "workspace:*",
    "@tanstack/react-query": "^5.59.0",
    "next": "^15.0.0",
    "react": "^19.0.0",
    "react-dom": "^19.0.0",
    "zustand": "^5.0.0"
  },
  "devDependencies": {
    "@speedplus/config": "workspace:*",
    "@types/node": "^22.0.0",
    "@types/react": "^19.0.0",
    "@types/react-dom": "^19.0.0",
    "tailwindcss": "^4.0.0",
    "@tailwindcss/postcss": "^4.0.0"
  }
}
```

- [ ] **Step 2: Create apps/merchant/package.json**

```json
{
  "name": "@speedplus/merchant",
  "version": "0.0.1",
  "private": true,
  "scripts": {
    "dev": "next dev --turbopack --port 3002",
    "build": "next build",
    "start": "next start",
    "typecheck": "tsc --noEmit"
  },
  "dependencies": {
    "@speedplus/ui": "workspace:*",
    "@speedplus/types": "workspace:*",
    "@speedplus/api-client": "workspace:*",
    "@speedplus/utils": "workspace:*",
    "@tanstack/react-query": "^5.59.0",
    "next": "^15.0.0",
    "react": "^19.0.0",
    "react-dom": "^19.0.0",
    "zustand": "^5.0.0",
    "react-hook-form": "^7.53.0",
    "zod": "^3.23.0",
    "@hookform/resolvers": "^3.9.0"
  },
  "devDependencies": {
    "@speedplus/config": "workspace:*",
    "@types/node": "^22.0.0",
    "@types/react": "^19.0.0",
    "@types/react-dom": "^19.0.0",
    "tailwindcss": "^4.0.0",
    "@tailwindcss/postcss": "^4.0.0"
  }
}
```

- [ ] **Step 3: Create apps/admin/package.json**

```json
{
  "name": "@speedplus/admin",
  "version": "0.0.1",
  "private": true,
  "scripts": {
    "dev": "next dev --turbopack --port 3003",
    "build": "next build",
    "start": "next start",
    "typecheck": "tsc --noEmit"
  },
  "dependencies": {
    "@speedplus/ui": "workspace:*",
    "@speedplus/types": "workspace:*",
    "@speedplus/api-client": "workspace:*",
    "@speedplus/utils": "workspace:*",
    "@tanstack/react-query": "^5.59.0",
    "next": "^15.0.0",
    "react": "^19.0.0",
    "react-dom": "^19.0.0",
    "zustand": "^5.0.0"
  },
  "devDependencies": {
    "@speedplus/config": "workspace:*",
    "@types/node": "^22.0.0",
    "@types/react": "^19.0.0",
    "@types/react-dom": "^19.0.0",
    "tailwindcss": "^4.0.0",
    "@tailwindcss/postcss": "^4.0.0"
  }
}
```

- [ ] **Step 4: Create minimal app shells for driver, merchant, admin**

For each of `apps/driver`, `apps/merchant`, `apps/admin` create identical minimal files:

`tsconfig.json` — same as customer but adjust paths
`next.config.ts` — same as customer (transpilePackages)
`postcss.config.mjs` — identical to customer
`.env.local` — `NEXT_PUBLIC_API_URL=http://localhost:8000/api/v1`
`app/globals.css` — identical to customer
`app/layout.tsx` — same as customer layout
`app/providers.tsx` — identical to customer
`app/page.tsx`:

```tsx
// apps/driver/app/page.tsx
import { SpeedPlusLogo } from '@speedplus/ui';
export default function DriverHome() {
  return (
    <main className="min-h-screen bg-midnight flex flex-col items-center justify-center">
      <SpeedPlusLogo variant="full" theme="dark" size="lg" />
      <p className="mt-4 text-primary text-sm">Driver Portal — Coming Soon</p>
    </main>
  );
}
```

Repeat for merchant (port 3002) and admin (port 3003) with appropriate labels.

- [ ] **Step 5: Final install and typecheck**

```bash
cd /home/obeej/Projects/speedplus
bun install
bunx turbo typecheck
```

Expected: All packages and apps typecheck clean.

- [ ] **Step 6: Commit**

```bash
git add apps/driver apps/merchant apps/admin
git commit -m "feat: scaffold driver, merchant and admin app shells"
```

---

## Verification

After all tasks complete:

```bash
# All apps run simultaneously
bun dev

# Customer:  http://localhost:3000
# Driver:    http://localhost:3001
# Merchant:  http://localhost:3002
# Admin:     http://localhost:3003
```

Expected on customer app:
- Dark midnight background fills viewport
- SpeedPlus logo (`Speed+` in spearmint) renders top-left
- "Faster. Cheaper. Better." tagline in `primary-300`
- Four vertical cards (Gas 🔥, Grocery 🛒, Food 🍽️, Pharmacy 💊) in 2-column grid
- No TypeScript errors in any app or package
