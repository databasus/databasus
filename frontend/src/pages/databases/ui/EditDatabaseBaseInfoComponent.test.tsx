// @vitest-environment happy-dom
import { screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

import { DatabaseType } from '@/entities/databases';
import { EditDatabaseBaseInfoComponent } from '@/pages/databases/ui/EditDatabaseBaseInfoComponent';
import { renderScreen } from '@/shared/lib/test/renderScreen';

describe('EditDatabaseBaseInfoComponent', () => {
  it('chooses an engine from the keyboard and announces its name once', async () => {
    const onContinue = vi.fn();
    const { user } = renderScreen(
      <EditDatabaseBaseInfoComponent
        name="Shop"
        type={DatabaseType.POSTGRES_LOGICAL}
        continueButtonText="Continue"
        onContinue={onContinue}
      />,
    );
    const mysqlCard = screen.getByRole('button', { name: 'MySQL' });

    expect(mysqlCard.getAttribute('aria-pressed')).toBe('false');

    mysqlCard.focus();
    await user.keyboard('{Enter}');

    expect(mysqlCard.getAttribute('aria-pressed')).toBe('true');
    expect(screen.getByRole('button', { name: 'PostgreSQL' }).getAttribute('aria-pressed')).toBe(
      'false',
    );

    await user.click(screen.getByRole('button', { name: 'Continue' }));

    expect(onContinue).toHaveBeenCalledWith('Shop', DatabaseType.MYSQL);
  });

  it('connects the name label to its field', () => {
    renderScreen(
      <EditDatabaseBaseInfoComponent
        name="Shop"
        type={DatabaseType.MYSQL}
        continueButtonText="Continue"
        onContinue={() => {}}
      />,
    );

    expect(screen.getByRole<HTMLInputElement>('textbox', { name: 'Name' }).value).toBe('Shop');
  });
});
