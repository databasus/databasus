// @vitest-environment happy-dom
import { act, renderHook, waitFor } from '@testing-library/react';
import { StrictMode, type ReactNode } from 'react';
import { type MockInstance, afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { DatabaseType, createDefaultDatabaseSettings, databaseApi } from '@/entities/databases';
import { useDatabaseConnectionTest } from '@/features/database-connection/model/useDatabaseConnectionTest';
import { activateSession, invalidateSession } from '@/shared/auth';
import { buildProblem, mockApi, setUpMockApi } from '@/shared/lib/test/mockApi';
import { releaseAndSettle } from '@/shared/lib/test/releaseAndSettle';
import { ScreenTestProviders } from '@/shared/lib/test/ScreenTestProviders';

const mockServer = setUpMockApi();

const target = {
  workspaceId: 'workspace-id',
  settings: createDefaultDatabaseSettings(DatabaseType.MYSQL),
};

const holdTest = (answer: { isConnected: boolean; error?: string }) => {
  const heldTest = mockApi.delayed({
    method: 'post',
    path: '/database-tests',
    status: 200,
    body: answer,
  });
  mockServer.use(heldTest.handler);

  return heldTest;
};

const countTestRequests = () =>
  mockServer.getRequests().filter((request) => request.url.endsWith('/database-tests')).length;

const renderTestHook = () => {
  const onChange = vi.fn();
  const onTestSucceeded = vi.fn();
  const hook = renderHook(() => useDatabaseConnectionTest<string>({ onChange, onTestSucceeded }), {
    wrapper: ScreenTestProviders,
  });

  return { ...hook, onChange, onTestSucceeded };
};

const startTest = async (hook: ReturnType<typeof renderTestHook>) => {
  act(() => {
    void hook.result.current.testConnection(target);
  });
  await waitFor(() => expect(countTestRequests()).toBeGreaterThan(0));
};

describe('useDatabaseConnectionTest', () => {
  let testCall: MockInstance;

  beforeEach(() => {
    activateSession({ token: 'test-token', userId: 'test-user' });
    testCall = vi.spyOn(databaseApi, 'testDatabaseSettings');
  });

  afterEach(() => {
    vi.restoreAllMocks();
    invalidateSession();
  });

  it('approves the settings after a successful test', async () => {
    const heldTest = holdTest({ isConnected: true });
    const hook = renderTestHook();

    await startTest(hook);

    expect(hook.result.current.isTesting).toBe(true);

    await releaseAndSettle(testCall, heldTest.release);

    expect(hook.result.current.isTested).toBe(true);
    expect(hook.result.current.isTesting).toBe(false);
    expect(hook.onTestSucceeded).toHaveBeenCalledTimes(1);
  });

  it('ignores a late success after a connection change', async () => {
    const heldTest = holdTest({ isConnected: true });
    const hook = renderTestHook();
    await startTest(hook);

    act(() => hook.result.current.changeConnection('new password'));
    await releaseAndSettle(testCall, heldTest.release);

    expect(hook.result.current.isTested).toBe(false);
    expect(hook.result.current.isTesting).toBe(false);
    expect(hook.onTestSucceeded).not.toHaveBeenCalled();
    expect(hook.onChange).toHaveBeenCalledWith('new password');
  });

  it('ignores a late failure after a connection change', async () => {
    const heldTest = holdTest({ isConnected: false, error: 'late failure' });
    const hook = renderTestHook();
    await startTest(hook);

    act(() => hook.result.current.changeConnection('new password'));
    await releaseAndSettle(testCall, heldTest.release);

    expect(hook.result.current.failure).toBeUndefined();
  });

  it('withdraws approval after a change away and back', async () => {
    const heldTest = holdTest({ isConnected: true });
    const hook = renderTestHook();
    await startTest(hook);
    await releaseAndSettle(testCall, heldTest.release);

    act(() => hook.result.current.changeConnection('5433'));
    act(() => hook.result.current.changeConnection('5432'));

    expect(hook.result.current.isTested).toBe(false);
  });

  it('removes the failure of a test after a connection change', async () => {
    const heldTest = holdTest({ isConnected: false, error: 'refused' });
    const hook = renderTestHook();
    await startTest(hook);
    await releaseAndSettle(testCall, heldTest.release);

    expect(hook.result.current.failure).toEqual({
      kind: 'result',
      result: { isConnected: false, error: 'refused' },
    });

    act(() => hook.result.current.changeConnection('other host'));

    expect(hook.result.current.failure).toBeUndefined();
  });

  it('keeps approval after a change that does not affect the connection', async () => {
    const heldTest = holdTest({ isConnected: true });
    const hook = renderTestHook();
    await startTest(hook);
    await releaseAndSettle(testCall, heldTest.release);

    act(() => hook.result.current.changeNonConnection('excluded table'));

    expect(hook.result.current.isTested).toBe(true);
    expect(hook.onChange).toHaveBeenCalledWith('excluded table');
  });

  it('sends one request when a test is started twice while pending', async () => {
    const heldTest = holdTest({ isConnected: true });
    const hook = renderTestHook();

    await startTest(hook);
    act(() => {
      void hook.result.current.testConnection(target);
    });
    await releaseAndSettle(testCall, heldTest.release);

    expect(countTestRequests()).toBe(1);
  });

  it('starts a new test after a change during a pending one and accepts only its answer', async () => {
    const firstTest = holdTest({ isConnected: true });
    const hook = renderTestHook();
    await startTest(hook);
    act(() => hook.result.current.changeConnection('new password'));
    const secondTest = holdTest({ isConnected: false, error: 'second answer' });

    act(() => {
      void hook.result.current.testConnection(target);
    });
    await waitFor(() => expect(countTestRequests()).toBe(2));
    await releaseAndSettle(testCall, firstTest.release, 0);

    expect(hook.result.current.isTested).toBe(false);
    expect(hook.result.current.isTesting).toBe(true);

    await releaseAndSettle(testCall, secondTest.release);

    expect(hook.result.current.failure).toEqual({
      kind: 'result',
      result: { isConnected: false, error: 'second answer' },
    });
    expect(hook.result.current.isTesting).toBe(false);
  });

  it('calls nothing when the answer arrives after unmount', async () => {
    const heldTest = holdTest({ isConnected: true });
    const hook = renderTestHook();
    await startTest(hook);

    hook.unmount();
    await releaseAndSettle(testCall, heldTest.release);

    expect(hook.onTestSucceeded).not.toHaveBeenCalled();
  });

  it('offers no approval when the session changes during a test', async () => {
    const heldTest = holdTest({ isConnected: true });
    const hook = renderTestHook();
    await startTest(hook);

    act(() => {
      invalidateSession();
    });
    await releaseAndSettle(testCall, heldTest.release);

    expect(hook.result.current.isTested).toBe(false);
    expect(hook.result.current.isTesting).toBe(false);
    expect(hook.onTestSucceeded).not.toHaveBeenCalled();
  });

  it('accepts a test started after the strict mode setup, cleanup and setup', async () => {
    const heldTest = holdTest({ isConnected: true });
    const onTestSucceeded = vi.fn();
    const { result } = renderHook(
      () => useDatabaseConnectionTest<string>({ onChange: () => {}, onTestSucceeded }),
      {
        wrapper: ({ children }: { children: ReactNode }) => (
          <StrictMode>
            <ScreenTestProviders>{children}</ScreenTestProviders>
          </StrictMode>
        ),
      },
    );

    act(() => {
      void result.current.testConnection(target);
    });
    await waitFor(() => expect(countTestRequests()).toBe(1));
    await releaseAndSettle(testCall, heldTest.release);

    expect(result.current.isTested).toBe(true);
    expect(onTestSucceeded).toHaveBeenCalledTimes(1);
  });

  it('shows no failure when a request rejected after the connection changed', async () => {
    const rejectedTest = mockApi.delayed({
      method: 'post',
      path: '/database-tests',
      status: 500,
      body: buildProblem('internal', 500),
    });
    mockServer.use(rejectedTest.handler);
    const hook = renderTestHook();

    await startTest(hook);
    act(() => hook.result.current.changeConnection('new password'));
    await releaseAndSettle(testCall, rejectedTest.release);

    expect(hook.result.current.failure).toBeUndefined();
    expect(hook.result.current.isTesting).toBe(false);
  });

  it('calls the latest success callback', async () => {
    const heldTest = holdTest({ isConnected: true });
    const firstCallback = vi.fn();
    const latestCallback = vi.fn();
    const { result, rerender } = renderHook(
      ({ onTestSucceeded }) =>
        useDatabaseConnectionTest<string>({ onChange: () => {}, onTestSucceeded }),
      { initialProps: { onTestSucceeded: firstCallback }, wrapper: ScreenTestProviders },
    );

    act(() => {
      void result.current.testConnection(target);
    });
    await waitFor(() => expect(countTestRequests()).toBe(1));
    rerender({ onTestSucceeded: latestCallback });
    await releaseAndSettle(testCall, heldTest.release);

    expect(firstCallback).not.toHaveBeenCalled();
    expect(latestCallback).toHaveBeenCalledTimes(1);
  });
});
