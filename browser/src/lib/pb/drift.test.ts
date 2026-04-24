// Drift check: loads the repo-wide golden fixture and asserts the TS wire
// codec roundtrips each case to byte-identical canonical JSON. If this test
// fails the TS implementation has drifted from shared/proto/testdata.
import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { fromJson, toJson } from './wire';

interface Case {
  name: string;
  message: Record<string, unknown>;
}

function loadCases(): Case[] {
  const p = resolve(__dirname, '../../../../shared/proto/testdata/messages.json');
  return JSON.parse(readFileSync(p, 'utf8')) as Case[];
}

function canonical(obj: unknown): string {
  // Deep sort keys so field order never matters.
  const sort = (v: unknown): unknown => {
    if (v === null || typeof v !== 'object') return v;
    if (Array.isArray(v)) return v.map(sort);
    return Object.keys(v as Record<string, unknown>)
      .sort()
      .reduce<Record<string, unknown>>((acc, k) => {
        acc[k] = sort((v as Record<string, unknown>)[k]);
        return acc;
      }, {});
  };
  return JSON.stringify(sort(obj));
}

describe('wire drift against shared/proto/testdata', () => {
  const cases = loadCases();
  expect(cases.length).toBeGreaterThan(0);

  for (const c of cases) {
    it(c.name, () => {
      const goldenJson = JSON.stringify(c.message);
      const parsed = fromJson(goldenJson);
      const produced = toJson(parsed);
      expect(canonical(JSON.parse(produced))).toBe(canonical(c.message));
    });
  }
});
