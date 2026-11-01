import { createHash } from 'node:crypto';
import { readFileSync } from 'node:fs';
import { describe, expect, it } from 'vitest';

describe('Theme assets', () => {
  it('ships licensed font assets and permits zoom without external font services', () => {
    const fontRoot = new URL('../../app/fonts/', import.meta.url);
    const source = JSON.parse(readFileSync(new URL('source.json', fontRoot), 'utf8')) as {
      assets: { file: string; bytes: number; sha256: string }[];
    };

    for (const asset of source.assets) {
      const bytes = readFileSync(new URL(asset.file, fontRoot));
      expect(bytes.length).toBe(asset.bytes);
      expect(createHash('sha256').update(bytes).digest('hex')).toBe(asset.sha256);
    }

    expect(readFileSync(new URL('LICENSE', fontRoot), 'utf8')).toContain('SIL OPEN FONT LICENSE');
    const html = readFileSync(new URL('../../../index.html', import.meta.url), 'utf8');
    expect(html).not.toMatch(/fonts\.google|maximum-scale|user-scalable=no/);
  });
});
