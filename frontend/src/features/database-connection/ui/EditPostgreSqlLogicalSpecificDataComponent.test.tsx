import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it, vi } from 'vitest';

import {
  DatabaseType,
  type PostgresqlLogicalSettings,
  PostgresSslMode,
  createDefaultDatabaseSettings,
} from '../../../entities/databases';
import { LocaleProvider } from '../../../shared/i18n';
import { EditPostgreSqlLogicalSpecificDataComponent } from './EditPostgreSqlLogicalSpecificDataComponent';

// The entity's public API imports its API client, and the client's config
// module reads window at import time. The tests run in node, so they supply
// window. LocaleProvider then reads localStorage, which here reports no saved
// language.
vi.hoisted(() => {
  vi.stubGlobal('window', {});
  vi.stubGlobal('localStorage', { getItem: () => null });
  vi.stubGlobal('navigator', { language: 'en-US', languages: ['en-US'] });
});

const createPostgresqlSettings = (
  change: Partial<PostgresqlLogicalSettings>,
): PostgresqlLogicalSettings => ({
  ...createDefaultDatabaseSettings(DatabaseType.POSTGRES_LOGICAL),
  version: '16',
  host: 'db.example.com',
  username: 'backup',
  database: 'orders',
  sslMode: PostgresSslMode.Require,
  ...change,
});

const renderPostgresqlForm = (settings: PostgresqlLogicalSettings) =>
  renderToStaticMarkup(
    <LocaleProvider>
      <EditPostgreSqlLogicalSpecificDataComponent
        database={{
          id: 'database-id',
          name: 'Orders',
          workspaceId: 'workspace-id',
          settings,
          notifierIds: [],
        }}
        settings={settings}
        onCancel={() => {}}
        isShowBackButton={false}
        onBack={() => {}}
        isSaveToApi={false}
        onSaved={() => {}}
      />
    </LocaleProvider>,
  );

describe('EditPostgreSqlLogicalSpecificDataComponent', () => {
  it('opens the advanced settings with both exclusion fields when only table data is excluded', () => {
    const markup = renderPostgresqlForm(createPostgresqlSettings({ excludeTableData: ['logs'] }));

    const wholeTablesLabelIndex = markup.indexOf('Exclude whole tables');
    const tableDataLabelIndex = markup.indexOf('Exclude table data');

    expect(wholeTablesLabelIndex).toBeGreaterThan(-1);
    expect(tableDataLabelIndex).toBeGreaterThan(wholeTablesLabelIndex);
    expect(markup.indexOf('logs', tableDataLabelIndex)).toBeGreaterThan(tableDataLabelIndex);
  });

  it('keeps the advanced settings collapsed when no advanced value is set', () => {
    const markup = renderPostgresqlForm(createPostgresqlSettings({}));

    expect(markup).toContain('Advanced settings');
    expect(markup).not.toContain('Exclude whole tables');
    expect(markup).not.toContain('Exclude table data');
  });
});
