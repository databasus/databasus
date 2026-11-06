import { describe, expect, it } from 'vitest';

import { buildLogicalBackupConfig } from '@/pages/databases/model/buildLogicalBackupConfig.fixture';
import { createDefaultLogicalBackupConfig } from '@/pages/databases/model/createDefaultLogicalBackupConfig';
import { hasNonDefaultLogicalAdvancedValues } from '@/pages/databases/model/hasNonDefaultLogicalAdvancedValues';

const defaultConfig = createDefaultLogicalBackupConfig();

describe('hasNonDefaultLogicalAdvancedValues', () => {
  it('answers false for the defaults', () => {
    expect(hasNonDefaultLogicalAdvancedValues(defaultConfig.settings)).toBe(false);
  });

  it.each([
    { field: 'retry off', settings: { isRetryIfFailed: false } },
    { field: 'another number of tries', settings: { maxFailedTriesCount: 5 } },
  ])('answers true for $field', ({ settings }) => {
    expect(
      hasNonDefaultLogicalAdvancedValues(buildLogicalBackupConfig({}, settings).settings),
    ).toBe(true);
  });
});
