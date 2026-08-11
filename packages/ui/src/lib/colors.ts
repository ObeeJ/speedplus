// Hex constants for contexts that can't take Tailwind classes (SVG fill/stroke
// props, inline style). Mirrors packages/config/tailwind/tokens.css — keep in
// sync if a token value changes. Values without a `// token:` comment are
// intentional derived tints (e.g. badge chip backgrounds) with no exact token
// equivalent; centralized here instead of duplicated as inline literals.

export const iconColors = {
  lime: '#C6F24E', // token: --color-lime
  emerald: '#0A3D2C', // token: --color-emerald
  tile: '#E9F3D8', // token: --color-tile
  sand: '#F7F5EF', // token: --color-sand
  stroke: '#1C3A2E', // token: --color-icon-stroke
  accent: '#7BA05B', // token: --color-icon-accent
  amberAccent: '#D9A408', // derived: darker amber for badge icon accents
  amberBg: '#FFF7E6', // derived: light amber tint for badge chip backgrounds
  mutedAccent: '#9A968D', // derived: muted gray for unknown/fallback badge icons
  mid: '#63636E', // token: --color-mid
  amberDeep: '#8A6A1B', // derived: deeper amber for icon-on-tint contrast
  indigo: '#3730A3', // derived: categorical accent (merchant stat icon)
  danger: '#B4231F', // derived: destructive-state icon accent
  dangerAlt: '#DC2626', // derived: destructive-state icon accent (alt shade)
  dangerBg: '#FEF2F2', // derived: light red tint for destructive-state chips
} as const;
