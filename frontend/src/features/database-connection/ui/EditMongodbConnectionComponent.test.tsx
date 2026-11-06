// @vitest-environment happy-dom
import { fireEvent, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';

import { DatabaseType, createDefaultDatabaseSettings } from '@/entities/databases';
import {
  buildCreateDraft,
  buildEditDraft,
  buildRestoreDraft,
  renderEditableForm,
  renderRestoreForm,
} from '@/features/database-connection/ui/connectionForm.fixture';
import { en } from '@/shared/i18n/locales/en';
import { setUpMockApi } from '@/shared/lib/test/mockApi';

setUpMockApi();

const mongodbSettings = {
  ...createDefaultDatabaseSettings(DatabaseType.MONGODB),
  host: 'db.example.com',
  username: 'backup',
  database: 'shop',
};

const openAdvanced = () => fireEvent.click(screen.getByText('Advanced settings'));

describe('EditMongodbConnectionComponent', () => {
  it('hides the port for an SRV connection', () => {
    renderEditableForm(buildCreateDraft({ ...mongodbSettings, isSrv: true, port: undefined }));

    expect(screen.queryByLabelText('Port')).toBeNull();
    expect(
      (screen.getByLabelText('Use SRV connection') as HTMLButtonElement).getAttribute(
        'aria-checked',
      ),
    ).toBe('true');
  });

  it('keeps the direct connection choice', () => {
    renderEditableForm(buildCreateDraft(mongodbSettings));
    openAdvanced();
    const directConnection = screen.getByLabelText('Direct connection');

    fireEvent.click(directConnection);

    expect(directConnection.getAttribute('aria-checked')).toBe('true');
  });

  it('turns SRV off and shows direct connection on when a tunnel is enabled', () => {
    renderEditableForm(buildCreateDraft({ ...mongodbSettings, isSrv: true }));

    fireEvent.click(screen.getByLabelText('SSH tunnel'));

    const srvSwitch = screen.getByLabelText('Use SRV connection') as HTMLButtonElement;
    const directConnection = screen.getByLabelText('Direct connection') as HTMLButtonElement;
    expect(srvSwitch.getAttribute('aria-checked')).toBe('false');
    expect(srvSwitch.disabled).toBe(true);
    expect(directConnection.getAttribute('aria-checked')).toBe('true');
    expect(directConnection.disabled).toBe(true);
    expect(screen.getByLabelText('Port')).toBeTruthy();
  });

  it('asks for the password again when the SRV flag of a saved database changes', () => {
    renderEditableForm(buildEditDraft({ ...mongodbSettings, password: undefined }));
    openAdvanced();

    fireEvent.click(screen.getByLabelText('Use SRV connection'));

    expect(screen.getByText(en.common.messages.addressChangedEnterSecretAgain)).toBeTruthy();
    expect(
      (screen.getByRole('button', { name: /Test connection$/ }) as HTMLButtonElement).disabled,
    ).toBe(true);
  });

  it('hides the collection exclusions in a restore target', () => {
    renderRestoreForm(buildRestoreDraft(mongodbSettings));
    openAdvanced();

    expect(screen.queryByText('Exclude collections')).toBeNull();
    expect(screen.getByLabelText('Use SRV connection')).toBeTruthy();
  });
});
