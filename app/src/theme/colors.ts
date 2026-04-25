// Color tokens.  The two scheme objects share the same key set so the
// rest of the codebase can hold one `Colors` type and not branch by mode.
//
// Source palette comes from product-direction sign-off in 2026-04 — keep
// in sync with docs/design.md when that lands.

export interface Colors {
  brandDark: string;       // primary action / focus / pressed
  brandLight: string;      // chips, badges, secondary fill
  brandPale: string;       // accent surface (hover, soft callouts)
  textPrimary: string;
  textSecondary: string;
  bgPage: string;          // outermost background
  bgCard: string;          // raised surface (cards, sheets, dropdowns)
  bgInput: string;         // form fields, inert chips
  borderDefault: string;
  // Semantic colors used by Badge / status pills.  Light values are
  // muted; dark values are slightly punchier so badges still pop on the
  // deeper backgrounds.
  semanticSuccess: string;
  semanticWarning: string;
  semanticDanger: string;
  semanticInfo: string;
}

export const lightColors: Colors = {
  brandDark:       '#0B505A',
  brandLight:      '#6FC0CC',
  brandPale:       '#E9F4F5',
  textPrimary:     '#152127',
  textSecondary:   '#557280',
  bgPage:          '#F2F6F6',
  bgCard:          '#FFFFFF',
  bgInput:         '#E9EFF0',
  borderDefault:   '#DAE4E6',
  semanticSuccess: '#1F7A4D',
  semanticWarning: '#A65A0E',
  semanticDanger:  '#B23A3A',
  semanticInfo:    '#155E96',
};

export const darkColors: Colors = {
  brandDark:       '#4FB0BE',
  brandLight:      '#167C8C',
  brandPale:       '#14353B',
  textPrimary:     '#E8F0F2',
  textSecondary:   '#9AB0B8',
  bgPage:          '#0B1418',
  bgCard:          '#152127',
  bgInput:         '#1E2D33',
  borderDefault:   '#243339',
  semanticSuccess: '#5BD391',
  semanticWarning: '#F0B45A',
  semanticDanger:  '#F08585',
  semanticInfo:    '#7CB6E5',
};
