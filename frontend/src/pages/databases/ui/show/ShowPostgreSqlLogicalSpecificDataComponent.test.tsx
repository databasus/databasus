import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it, vi } from 'vitest';

import {
  DatabaseType,
  type PostgresqlLogicalSettings,
  PostgresSslMode,
  createDefaultDatabaseSettings,
} from '../../../../entities/databases';
import { LocaleProvider } from '../../../../shared/i18n';
import { ShowPostgreSqlLogicalSpecificDataComponent } from './ShowPostgreSqlLogicalSpecificDataComponent';

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

const renderPostgresqlPage = (settings: PostgresqlLogicalSettings) =>
  renderToStaticMarkup(
    <LocaleProvider>
      <ShowPostgreSqlLogicalSpecificDataComponent settings={settings} />
    </LocaleProvider>,
  );

describe('ShowPostgreSqlLogicalSpecificDataComponent', () => {
  it('shows each exclusion list under its own label', () => {
    const markup = renderPostgresqlPage(
      createPostgresqlSettings({ excludeTables: ['audit_tmp'], excludeTableData: ['logs'] }),
    );

    const wholeTablesLabelIndex = markup.indexOf('Exclude whole tables');
    const tableDataLabelIndex = markup.indexOf('Exclude table data');

    expect(wholeTablesLabelIndex).toBeGreaterThan(-1);
    expect(markup.indexOf('audit_tmp')).toBeGreaterThan(wholeTablesLabelIndex);
    expect(markup.indexOf('audit_tmp')).toBeLessThan(tableDataLabelIndex);
    expect(markup.indexOf('logs')).toBeGreaterThan(tableDataLabelIndex);
  });

  it('hides both exclusion rows when neither list is set', () => {
    const markup = renderPostgresqlPage(createPostgresqlSettings({}));

    expect(markup).not.toContain('Exclude whole tables');
    expect(markup).not.toContain('Exclude table data');
  });
});
