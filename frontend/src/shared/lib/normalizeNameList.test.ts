import { describe, expect, it } from 'vitest';

import { normalizeNameList } from '@/shared/lib/normalizeNameList';

describe('normalizeNameList', () => {
  it('trims separators glued to pasted names', () => {
    expect(normalizeNameList(['personnel_access_control_event', '\npersonnel_real_time'])).toEqual([
      'personnel_access_control_event',
      'personnel_real_time',
    ]);
  });

  it('splits entries that still contain separators', () => {
    expect(normalizeNameList(['orders,\nusers\tpayments'])).toEqual([
      'orders',
      'users',
      'payments',
    ]);
  });

  it('drops blank entries and duplicates', () => {
    expect(normalizeNameList([' orders ', '', '   ', 'orders'])).toEqual(['orders']);
  });
});
