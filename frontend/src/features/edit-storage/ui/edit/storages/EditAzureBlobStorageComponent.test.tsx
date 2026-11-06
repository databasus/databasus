// @vitest-environment happy-dom
import { screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';

import { EditAzureBlobStorageComponent } from '@/features/edit-storage/ui/edit/storages/EditAzureBlobStorageComponent';
import { renderScreen } from '@/shared/lib/test/renderScreen';

describe('EditAzureBlobStorageComponent', () => {
  it('names the authentication method choice by its label', () => {
    renderScreen(
      <EditAzureBlobStorageComponent
        settings={{ type: 'AZURE_BLOB', authMethod: 'ACCOUNT_KEY', containerName: 'backups' }}
        isSaved={false}
        isAddressChanged={false}
        onChange={() => {}}
      />,
    );

    expect(screen.getByRole('radiogroup', { name: 'Auth method' })).toBeTruthy();
  });
});
