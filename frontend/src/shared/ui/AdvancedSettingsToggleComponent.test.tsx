// @vitest-environment happy-dom
import { screen } from '@testing-library/react';
import { useState } from 'react';
import { describe, expect, it } from 'vitest';

import { renderScreen } from '../lib/test/renderScreen';
import { AdvancedSettingsToggleComponent } from './AdvancedSettingsToggleComponent';

const Disclosure = () => {
  const [isExpanded, setIsExpanded] = useState(false);
  return (
    <>
      <AdvancedSettingsToggleComponent
        label="Advanced"
        isShowAdvanced={isExpanded}
        regionId="advanced-fields"
        onToggle={() => setIsExpanded(!isExpanded)}
      />
      {isExpanded && <div id="advanced-fields">Fields</div>}
    </>
  );
};

describe('AdvancedSettingsToggleComponent', () => {
  it('updates expanded state when a disclosure is activated from the keyboard', async () => {
    const { user } = renderScreen(<Disclosure />);
    const disclosure = screen.getByRole('button', { name: 'Advanced' });
    disclosure.focus();

    await user.keyboard('{Enter}');
    expect(disclosure.getAttribute('aria-expanded')).toBe('true');
    expect(document.getElementById(disclosure.getAttribute('aria-controls')!)).not.toBeNull();

    await user.keyboard(' ');
    expect(disclosure.getAttribute('aria-expanded')).toBe('false');
  });
});
