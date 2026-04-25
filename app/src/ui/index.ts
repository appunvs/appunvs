// Public surface for the design-system primitives.  Importers should
// pull from `@/ui` rather than reaching into individual files; this
// makes future swaps (e.g. wrapping Pressable in a haptics layer)
// invisible to callers.

export { Text }       from './Text';
export type { TextProps } from './Text';

export { Button }     from './Button';
export type { ButtonProps } from './Button';

export { Card }       from './Card';
export type { CardProps } from './Card';

export { Input }      from './Input';
export type { InputProps } from './Input';

export { IconButton } from './IconButton';
export type { IconButtonProps } from './IconButton';

export { Divider }    from './Divider';

export { Badge }      from './Badge';
export type { BadgeProps, BadgeTone } from './Badge';
