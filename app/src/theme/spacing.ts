// Spacing scale.  Powers of 2 + a 12 step where 8/16 are too far apart.
// Use named keys at call sites — `spacing.m` is more grep-able than `12`.
export const spacing = {
  none: 0,
  xs:   4,
  s:    8,
  m:    12,
  l:    16,
  xl:   20,
  xxl:  24,
  xxxl: 32,
  huge: 48,
} as const;

export type SpacingKey = keyof typeof spacing;
