// @vitest-environment happy-dom
import { screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

import { DatabaseType } from '@/entities/databases';
import { ChoosePostgresBackupTypeComponent } from '@/pages/databases/create-wizard/ui/ChoosePostgresBackupTypeComponent';
import { DICTIONARIES } from '@/shared/i18n/dictionaries';
import { renderScreen } from '@/shared/lib/test/renderScreen';

const { logical, physical } = DICTIONARIES.en.databases.create.backupType;

describe('ChoosePostgresBackupTypeComponent', () => {
  it('chooses a backup type from the keyboard and exposes the choice', async () => {
    const onSelected = vi.fn();
    const { user } = renderScreen(
      <ChoosePostgresBackupTypeComponent
        type={DatabaseType.POSTGRES_LOGICAL}
        continueButtonText="Continue"
        onBack={() => {}}
        onSelected={onSelected}
      />,
    );
    const logicalCard = screen.getByRole('button', { name: new RegExp(`^${logical.title}`) });
    const physicalCard = screen.getByRole('button', { name: new RegExp(`^${physical.title}`) });

    expect(logicalCard.getAttribute('aria-pressed')).toBe('true');
    expect(physicalCard.getAttribute('aria-pressed')).toBe('false');

    physicalCard.focus();
    await user.keyboard(' ');

    expect(logicalCard.getAttribute('aria-pressed')).toBe('false');
    expect(physicalCard.getAttribute('aria-pressed')).toBe('true');

    await user.click(screen.getByRole('button', { name: 'Continue' }));

    expect(onSelected).toHaveBeenCalledWith(DatabaseType.POSTGRES_PHYSICAL);
  });
});
