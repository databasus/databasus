import { cleanup } from '@testing-library/react';
import dayjs from 'dayjs';
import { afterEach, beforeEach, vi } from 'vitest';

const readStorage = (storage: Storage) =>
  Array.from({ length: storage.length }, (_, index) => {
    const key = storage.key(index)!;
    return [key, storage.getItem(key)!] as const;
  });
const restoreStorage = (storage: Storage, entries: ReadonlyArray<readonly [string, string]>) => {
  storage.clear();
  entries.forEach(([key, value]) => storage.setItem(key, value));
};

let restoreDocument = () => {};
beforeEach(() => {
  const localEntries = readStorage(localStorage);
  const sessionEntries = readStorage(sessionStorage);
  const address = window.location.href;
  const historyState: unknown = window.history.state;
  const title = document.title;
  const language = document.documentElement.getAttribute('lang');
  const classes = document.documentElement.className;
  const colorScheme = document.documentElement.getAttribute('data-mantine-color-scheme');
  const reducedMotion = document.documentElement.getAttribute('data-respect-reduced-motion');
  const dayjsLocale = dayjs.locale();

  restoreDocument = () => {
    restoreStorage(localStorage, localEntries);
    restoreStorage(sessionStorage, sessionEntries);
    window.history.replaceState(historyState, '', address);
    document.title = title;
    if (language === null) document.documentElement.removeAttribute('lang');
    else document.documentElement.lang = language;
    document.documentElement.className = classes;
    if (colorScheme === null) document.documentElement.removeAttribute('data-mantine-color-scheme');
    else document.documentElement.setAttribute('data-mantine-color-scheme', colorScheme);
    if (reducedMotion === null)
      document.documentElement.removeAttribute('data-respect-reduced-motion');
    else document.documentElement.setAttribute('data-respect-reduced-motion', reducedMotion);
    dayjs.locale(dayjsLocale);
  };
});

afterEach(() => {
  cleanup();
  if (vi.isFakeTimers()) vi.clearAllTimers();
  vi.useRealTimers();
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
  restoreDocument();
});
