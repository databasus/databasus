// @vitest-environment happy-dom
import { afterEach, describe, expect, it, vi } from 'vitest';

import {
  activateSession,
  beginAuthIntent,
  abandonAuthIntent,
  acceptAuthSession,
  getSessionSnapshot,
  invalidateSession,
  subscribeSession,
} from '@/shared/auth/session';

afterEach(() => {
  invalidateSession();
  localStorage.clear();
  sessionStorage.clear();
});

describe('Session', () => {
  it('keeps other subscriptions active when a disposer is repeated', () => {
    const firstObserver = vi.fn();
    const secondObserver = vi.fn();
    const disposeFirst = subscribeSession(firstObserver);
    const disposeSecond = subscribeSession(secondObserver);

    disposeFirst();
    disposeFirst();
    activateSession({ token: 'current', userId: 'owner' });

    expect(firstObserver).not.toHaveBeenCalled();
    expect(secondObserver).toHaveBeenCalledOnce();
    disposeSecond();
  });

  it('publishes one coherent transition when the active session is refused repeatedly', () => {
    activateSession({ token: 'first', userId: 'first-owner' });
    const firstSession = getSessionSnapshot();
    const observer = vi.fn();
    const dispose = subscribeSession(observer);

    expect(invalidateSession(firstSession)).toBe(true);
    expect(invalidateSession(firstSession)).toBe(false);
    expect(observer).toHaveBeenCalledOnce();
    expect(getSessionSnapshot()).toMatchObject({ token: undefined, userId: undefined });

    dispose();
  });

  it('preserves preferences and clears account selection when a session is replaced', () => {
    activateSession({ token: 'first', userId: 'first-owner' });
    const firstSession = getSessionSnapshot();
    localStorage.setItem('selected_workspace_id', 'old-workspace');
    localStorage.setItem('selectedDatabaseId_old-workspace', 'old-database');
    localStorage.setItem('selectedStorageId_old-workspace', 'old-storage');
    localStorage.setItem('selectedNotifierId_old-workspace', 'old-notifier');
    localStorage.setItem('logicalBackupListFilter', '{"statuses":["FAILED"]}');
    localStorage.setItem('physicalBackupsFilters', '{"types":["FULL"]}');
    localStorage.setItem('databasus-theme', 'dark');
    localStorage.setItem('databasus-locale', 'fr');
    localStorage.setItem('lastVersionReload', '123');
    sessionStorage.setItem('databasus_oauth_sign_in_attempt', 'old-attempt');

    activateSession({ token: 'second', userId: 'second-owner' });
    expect(invalidateSession(firstSession)).toBe(false);

    expect(localStorage.getItem('selected_workspace_id')).toBeNull();
    expect(localStorage.getItem('selectedDatabaseId_old-workspace')).toBeNull();
    expect(localStorage.getItem('selectedStorageId_old-workspace')).toBeNull();
    expect(localStorage.getItem('selectedNotifierId_old-workspace')).toBeNull();
    expect(localStorage.getItem('logicalBackupListFilter')).toBeNull();
    expect(localStorage.getItem('physicalBackupsFilters')).toBeNull();
    expect(sessionStorage.getItem('databasus_oauth_sign_in_attempt')).toBeNull();
    expect(localStorage.getItem('databasus-theme')).toBe('dark');
    expect(localStorage.getItem('databasus-locale')).toBe('fr');
    expect(localStorage.getItem('lastVersionReload')).toBe('123');
  });

  it('refuses abandoned completions when another intent or session takes ownership', () => {
    const abandoned = beginAuthIntent();
    abandonAuthIntent(abandoned);
    expect(
      acceptAuthSession({ intent: abandoned, session: { token: 'stale', userId: 'stale' } }),
    ).toBe(false);

    const replaced = beginAuthIntent();
    const current = beginAuthIntent();
    expect(
      acceptAuthSession({ intent: replaced, session: { token: 'stale', userId: 'stale' } }),
    ).toBe(false);

    expect(
      acceptAuthSession({ intent: current, session: { token: 'current', userId: 'owner' } }),
    ).toBe(true);
    expect(
      acceptAuthSession({ intent: current, session: { token: 'repeated', userId: 'owner' } }),
    ).toBe(false);
  });

  it('resubscribes successfully after setup cleanup and setup', () => {
    const observer = vi.fn();
    const dispose = subscribeSession(observer);
    dispose();
    const disposeAgain = subscribeSession(observer);

    activateSession({ token: 'current', userId: 'owner' });
    expect(observer).toHaveBeenCalledOnce();

    disposeAgain();
  });
});
