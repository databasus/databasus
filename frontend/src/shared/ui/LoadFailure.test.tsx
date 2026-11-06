// @vitest-environment happy-dom
import { screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

import { ApiError } from '@/shared/api';
import { DICTIONARIES } from '@/shared/i18n';
import { renderScreen } from '@/shared/lib/test/renderScreen';
import { LoadFailure } from '@/shared/ui/LoadFailure';

describe('LoadFailure', () => {
  it('shows the translated error and retries on click', async () => {
    const onRetry = vi.fn();
    const { user } = renderScreen(
      <LoadFailure error={new ApiError({ code: 'internal', status: 500 })} onRetry={onRetry} />,
      { locale: 'fr' },
    );

    expect(screen.getByText(DICTIONARIES.fr.errors.internal)).toBeTruthy();

    await user.click(screen.getByRole('button', { name: DICTIONARIES.fr.common.actions.retry }));

    expect(onRetry).toHaveBeenCalledTimes(1);
  });
});
