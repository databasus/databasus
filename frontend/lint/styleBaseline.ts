import type { StyleFinding } from './styleChecks.ts';

export interface StyleBaselineEntry extends StyleFinding {
  count: number;
  owner: string;
  removalMilestone: string;
}
const getIdentity = (finding: StyleFinding) =>
  JSON.stringify([finding.rule, finding.source, finding.construct]);

export const compareStyleBaseline = ({
  findings,
  baseline,
}: {
  findings: StyleFinding[];
  baseline: StyleBaselineEntry[];
}): string[] => {
  const counts = new Map<string, number>();
  for (const finding of findings)
    counts.set(getIdentity(finding), (counts.get(getIdentity(finding)) ?? 0) + 1);

  const errors: string[] = [];
  const seen = new Set<string>();
  for (const entry of baseline) {
    const identity = getIdentity(entry);

    if (
      seen.has(identity) ||
      !entry.owner ||
      !entry.removalMilestone ||
      !Number.isInteger(entry.count) ||
      entry.count < 1
    )
      errors.push(`Invalid style baseline: ${identity}`);
    seen.add(identity);

    const count = counts.get(identity) ?? 0;
    if (count !== entry.count)
      errors.push(`Style baseline count changed (${entry.count} -> ${count}): ${identity}`);
    counts.delete(identity);
  }

  for (const [identity, count] of counts)
    errors.push(`New style violation (${count}): ${identity}`);
  return errors;
};
