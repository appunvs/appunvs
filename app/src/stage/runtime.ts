// TypeScript-only re-export.  Metro's resolver prefers `runtime.native.tsx`
// / `runtime.web.tsx` over this file at runtime, so this is exclusively
// for the type-checker / IDE; it lets `import { StageRuntime } from '@/stage/runtime'`
// resolve without configuring tsconfig moduleSuffixes.
export { StageRuntime } from './runtime.native';
export type { StageRuntimeProps } from './runtime.types';
