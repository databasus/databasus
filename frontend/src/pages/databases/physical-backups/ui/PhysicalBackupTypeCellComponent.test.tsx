import type { JSX } from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it, vi } from 'vitest';

import { type PhysicalBackupListItem } from '@/pages/databases/physical-backups/model/PhysicalBackupListItem';
import { PhysicalBackupStatus } from '@/pages/databases/physical-backups/model/PhysicalBackupStatus';
import { PhysicalBackupType } from '@/pages/databases/physical-backups/model/PhysicalBackupType';
import {
  PhysicalBackupTypeCellComponent,
  WalRangeSegmentsComponent,
} from '@/pages/databases/physical-backups/ui/PhysicalBackupTypeCellComponent';
import { LocaleProvider } from '@/shared/i18n';
import { ThemeContext, ThemeConfiguration } from '@/shared/theme';

// The entity's public API imports its API client, and the client's config
// module reads window at import time. The tests run in node, so they supply
// window. LocaleProvider then reads localStorage, which here reports no saved
// language.
vi.hoisted(() => {
  vi.stubGlobal('window', {});
  vi.stubGlobal('localStorage', { getItem: () => null });
  vi.stubGlobal('navigator', { language: 'en-US', languages: ['en-US'] });
});

const FIRST_SEGMENT = '000000010000000000000010';
const LAST_SEGMENT = '0000000100000000000000FF';

const createWalRange = (change: Partial<PhysicalBackupListItem>): PhysicalBackupListItem => ({
  id: 'range-id',
  type: PhysicalBackupType.WAL,
  status: PhysicalBackupStatus.COMPLETED,
  timelineId: 1,
  startLsn: '0/10000000',
  stopLsn: '1/0',
  fromWalFilename: FIRST_SEGMENT,
  toWalFilename: LAST_SEGMENT,
  walSegmentsCount: 240,
  hasGapBefore: false,
  sizeMb: 3840,
  createdAt: new Date('2026-10-10T03:00:00Z').toISOString(),
  completedAt: new Date('2026-10-10T09:12:00Z').toISOString(),
  ...change,
});

const renderWithLocale = (element: JSX.Element) =>
  renderToStaticMarkup(
    <ThemeContext.Provider value={{ theme: 'light', resolvedTheme: 'light', setTheme: () => {} }}>
      <ThemeConfiguration isTestEnvironment>
        <LocaleProvider>{element}</LocaleProvider>
      </ThemeConfiguration>
    </ThemeContext.Provider>,
  );

describe('PhysicalBackupTypeCellComponent', () => {
  it('shows only the type badge for a WAL range', () => {
    const markup = renderWithLocale(
      <PhysicalBackupTypeCellComponent backup={createWalRange({})} />,
    );

    expect(markup).toContain('WAL');
    expect(markup).not.toContain(FIRST_SEGMENT);
    expect(markup).not.toContain('Segments:');
  });

  it('shows only the type badge for a backup', () => {
    const markup = renderWithLocale(
      <PhysicalBackupTypeCellComponent
        backup={createWalRange({
          type: PhysicalBackupType.FULL,
          fromWalFilename: undefined,
          toWalFilename: undefined,
          walSegmentsCount: undefined,
        })}
      />,
    );

    expect(markup).toContain('Full');
  });
});

describe('WalRangeSegmentsComponent', () => {
  it('shows the first and last segment of a range', () => {
    const markup = renderWithLocale(<WalRangeSegmentsComponent backup={createWalRange({})} />);

    expect(markup).toContain(FIRST_SEGMENT);
    expect(markup).toContain(LAST_SEGMENT);
  });

  it('shows one segment name for a single-segment range', () => {
    const markup = renderWithLocale(
      <WalRangeSegmentsComponent
        backup={createWalRange({ toWalFilename: FIRST_SEGMENT, walSegmentsCount: 1 })}
      />,
    );

    expect(markup.split(FIRST_SEGMENT)).toHaveLength(2);
  });
});
