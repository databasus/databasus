// @vitest-environment happy-dom
import { screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';

import { renderScreen } from '../lib/test/renderScreen';
import { AdvancedFieldRowComponent } from './AdvancedFieldRowComponent';

describe('AdvancedFieldRowComponent', () => {
  it('associates labels and help when a field declares its control ID', () => {
    renderScreen(
      <AdvancedFieldRowComponent controlId="host" label="Host" tooltip="SMTP host">
        <input id="host" aria-describedby="host-help" />
      </AdvancedFieldRowComponent>,
    );
    expect(screen.getByRole('textbox', { name: 'Host' }).getAttribute('aria-describedby')).toBe(
      'host-help',
    );
    expect(document.getElementById('host-help')).not.toBeNull();
  });
});
