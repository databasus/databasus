import { describe, expect, it } from 'vitest';

import { compareStyleBaseline } from '../lint/styleBaseline.ts';
import { checkStyleSource } from '../lint/styleChecks.ts';

const check = ({ content }: { content: string }) =>
  checkStyleSource({ content, source: 'shared/ui/Control.tsx' });

describe('Style baseline comparison', () => {
  it('tracks exact counts and rejects stale exceptions when source occurrences change', () => {
    const content = `<div style={{ color: 'red' }} />`;
    const findings = check({ content: content });
    const baseline = findings.map((finding) => ({
      ...finding,
      count: 1,
      owner: 'notifiers',
      removalMilestone: '4',
    }));
    expect(
      compareStyleBaseline({ findings: check({ content: `\n\n${content}` }), baseline }),
    ).toEqual([]);
    expect(
      compareStyleBaseline({ findings: check({ content: `${content};${content}` }), baseline })[0],
    ).toContain('1 -> 2');
    expect(compareStyleBaseline({ findings: [], baseline })[0]).toContain('1 -> 0');
    expect(compareStyleBaseline({ findings, baseline: [] })[0]).toContain('New style violation');
  });
});
