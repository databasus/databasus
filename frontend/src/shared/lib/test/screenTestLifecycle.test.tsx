import { fileURLToPath } from 'node:url';
import { describe, expect, it } from 'vitest';

import { runIsolatedScenario } from '@/shared/lib/test/runIsolatedScenario';

describe('screenTestLifecycle', () => {
  it('restores screen context even after a failing test', async () => {
    const child = await runIsolatedScenario({
      scenario: `
import { createElement } from 'react';
import { vi } from 'vitest';
import { renderScreen } from ${JSON.stringify(fileURLToPath(new URL('./renderScreen.tsx', import.meta.url)))};
const title = document.title;
const language = document.documentElement.lang;
const classes = document.documentElement.className;
it('deliberately fails after rendering', () => {
  vi.useFakeTimers();
  renderScreen(createElement('input'), { locale: 'fr', theme: 'dark' });
  vi.stubGlobal('fixtureGlobal', 'temporary');
  sessionStorage.setItem('fixture', 'temporary');
  throw new Error('deliberate fixture failure');
});
it('observes restored context after failure', () => {
  expect(document.title).toBe(title);
  expect(document.documentElement.lang).toBe(language);
  expect(document.documentElement.className).toBe(classes);
  expect(localStorage.length).toBe(0);
  expect(sessionStorage.length).toBe(0);
  expect('fixtureGlobal' in globalThis).toBe(false);
  expect(vi.isFakeTimers()).toBe(false);
  expect(document.body.childElementCount).toBe(0);
});`,
      environment: 'happy-dom',
    });
    expect(child.code).not.toBe(0);
    expect(child.output).toContain('deliberate fixture failure');
    expect(child.output).toContain('1 failed | 1 passed');
  });
});
