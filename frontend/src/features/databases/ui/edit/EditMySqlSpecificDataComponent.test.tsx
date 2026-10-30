import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it, vi } from 'vitest';

import { type Database, type MysqlSettings } from '../../../../entities/databases';
import { LocaleProvider } from '../../../../shared/i18n';
import { EditMySqlSpecificDataComponent } from './EditMySqlSpecificDataComponent';

// The entity's public API imports its API client, and the client's config
// module reads window at import time. The tests run in node, so they supply
// window. LocaleProvider then reads localStorage, which here reports no saved
// language.
vi.hoisted(() => {
  vi.stubGlobal('window', {});
  vi.stubGlobal('localStorage', { getItem: () => null });
  vi.stubGlobal('navigator', { language: 'en-US', languages: ['en-US'] });
});

const mysqlSettingsWithExcludedTable: MysqlSettings = {
  type: 'MYSQL',
  version: '8.4',
  host: 'db.example.com',
  port: 3306,
  username: 'backup',
  password: '',
  database: 'shop',
  isHttps: false,
  sshTunnel: { isEnabled: false },
  excludeTables: ['audit_log'],
};

const mysqlDatabaseWithExcludedTable: Database = {
  id: 'database-id',
  name: 'Shop',
  workspaceId: 'workspace-id',
  settings: mysqlSettingsWithExcludedTable,
  notifierIds: [],
};

describe('EditMySqlSpecificDataComponent', () => {
  it('keeps the table exclusion label that has no data-only counterpart', () => {
    const markup = renderToStaticMarkup(
      <LocaleProvider>
        <EditMySqlSpecificDataComponent
          database={mysqlDatabaseWithExcludedTable}
          settings={mysqlSettingsWithExcludedTable}
          onCancel={() => {}}
          isShowBackButton={false}
          onBack={() => {}}
          isSaveToApi={false}
          onSaved={() => {}}
        />
      </LocaleProvider>,
    );

    expect(markup).toContain('Exclude tables');
    expect(markup).toContain('audit_log');
    expect(markup).not.toContain('Exclude whole tables');
  });
});
