// @vitest-environment happy-dom
import { fireEvent, screen, waitFor } from '@testing-library/react';
import { describe, expect, it } from 'vitest';

import {
  DatabaseType,
  type DatabaseSettings,
  PostgresSslMode,
  createDefaultDatabaseSettings,
} from '@/entities/databases';
import {
  buildCreateDraft,
  renderEditableForm,
} from '@/features/database-connection/ui/connectionForm.fixture';
import { setUpMockApi } from '@/shared/lib/test/mockApi';
import { stubClipboard } from '@/shared/lib/test/stubClipboard';

setUpMockApi();

const newPhysicalSettings = {
  ...createDefaultDatabaseSettings(DatabaseType.POSTGRES_PHYSICAL),
  version: '16',
  host: 'db.example.com',
  username: 'replicator',
  sslMode: PostgresSslMode.Require,
} as DatabaseSettings;

const getHostInput = () => screen.getByLabelText('Host') as HTMLInputElement;

const getSelectedSslMode = () =>
  screen.getByText('SSL mode').parentElement!.querySelector('.ant-select-selection-item')!
    .textContent;

describe('EditPostgresqlPhysicalConnectionComponent', () => {
  it('turns SSL off for a new database on localhost without a chosen mode', () => {
    renderEditableForm(buildCreateDraft(newPhysicalSettings));

    fireEvent.change(getHostInput(), { target: { value: 'localhost' } });

    expect(getSelectedSslMode()).toMatch(/^Disable/);
  });

  it('keeps an imported mode of a new database when the host then changes', async () => {
    renderEditableForm(buildCreateDraft(newPhysicalSettings));
    stubClipboard(
      async () => 'postgresql://importer:secret@db.example.com:5432/shop?sslmode=require',
    );

    fireEvent.click(screen.getByText('Parse from clipboard'));
    await waitFor(() => expect(screen.getByDisplayValue('importer')).toBeTruthy());
    fireEvent.change(getHostInput(), { target: { value: 'localhost' } });

    expect(getSelectedSslMode()).toMatch(/^Require/);
  });
});
