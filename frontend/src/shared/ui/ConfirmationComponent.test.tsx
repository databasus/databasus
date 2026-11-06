// @vitest-environment happy-dom
import { screen, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';

import { renderScreen } from '@/shared/lib/test/renderScreen';
import { ConfirmationComponent } from '@/shared/ui/ConfirmationComponent';

describe('ConfirmationComponent', () => {
  it('keeps cancellation ordinary when the confirmed action is destructive', () => {
    renderScreen(
      <ConfirmationComponent
        description="Remove channel?"
        actionText="Remove"
        isDestructive
        onConfirm={() => {}}
        onDecline={() => {}}
      />,
    );
    const dialog = within(screen.getByRole('dialog'));
    expect(
      dialog.getByRole('button', { name: 'Remove' }).classList.contains('ant-btn-dangerous'),
    ).toBe(true);
    expect(
      dialog.getByRole('button', { name: 'Cancel' }).classList.contains('ant-btn-dangerous'),
    ).toBe(false);
  });
});
