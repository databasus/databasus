import { describe, expect, it } from 'vitest';

import { quotePostgresqlIdentifier } from './quotePostgresqlIdentifier';

describe('quotePostgresqlIdentifier', () => {
  it('quotes a Databasus-generated role name containing a hyphen', () => {
    expect(quotePostgresqlIdentifier('databasus-ab12cd34')).toBe('"databasus-ab12cd34"');
  });

  it('escapes embedded double quotes', () => {
    expect(quotePostgresqlIdentifier('backup"role')).toBe('"backup""role"');
  });
});
