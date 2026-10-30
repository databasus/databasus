export interface ArchitectureFinding {
  rule: string;
  source: string;
  target: string;
}

export interface ArchitectureBaselineEntry extends ArchitectureFinding {
  count: number;
  owner: string;
  milestone: number;
}

export const getFindingIdentity = ({ rule, source, target }: ArchitectureFinding) =>
  JSON.stringify([rule, source, target]);

export const compareArchitectureBaseline = (
  findings: ArchitectureFinding[],
  entries: ArchitectureBaselineEntry[],
): string[] => {
  const counts = new Map<string, number>();
  for (const finding of findings) {
    const identity = getFindingIdentity(finding);
    counts.set(identity, (counts.get(identity) ?? 0) + 1);
  }

  const errors: string[] = [];
  const registered = new Set<string>();
  for (const entry of entries) {
    const identity = getFindingIdentity(entry);
    if (
      registered.has(identity) ||
      !entry.owner ||
      !Number.isInteger(entry.milestone) ||
      entry.milestone < 1 ||
      !Number.isInteger(entry.count) ||
      entry.count < 1
    ) {
      errors.push(`Invalid or duplicate baseline entry: ${identity}`);
    }
    registered.add(identity);
    const count = counts.get(identity) ?? 0;
    if (count !== entry.count) {
      errors.push(
        `${count > entry.count ? 'Increased' : 'Stale'} baseline finding: ${identity} (${entry.count} recorded, ${count} present)`,
      );
    }
    counts.delete(identity);
  }

  for (const [identity, count] of counts) {
    errors.push(`New architecture finding: ${identity} (${count} occurrences)`);
  }
  return errors;
};
