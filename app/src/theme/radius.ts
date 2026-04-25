// Corner radius scale.  Pillow-soft `xl` reserved for chat bubbles only;
// cards stay at `l`, inputs at `m`, badges/buttons at `m`.
export const radius = {
  none: 0,
  s:    6,
  m:    10,
  l:    12,
  xl:   14,
  pill: 999,
} as const;

export type RadiusKey = keyof typeof radius;
