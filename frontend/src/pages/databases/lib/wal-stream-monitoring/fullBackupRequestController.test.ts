import { describe, expect, it, vi } from 'vitest';

import {
  type FullBackupRequestDependencies,
  FullBackupRequestController,
} from './fullBackupRequestController';

const createDeferredRequest = () => {
  let resolveRequest: () => void = () => {};
  let rejectRequest: (error: unknown) => void = () => {};

  const promise = new Promise<void>((resolve, reject) => {
    resolveRequest = resolve;
    rejectRequest = reject;
  });

  return { promise, resolveRequest, rejectRequest };
};

const createRequestController = (requestFullBackup: () => Promise<unknown>) => {
  const dependencies: FullBackupRequestDependencies = {
    requestFullBackup: vi.fn(requestFullBackup),
    refreshNow: vi.fn(),
    reportRequested: vi.fn(),
    reportError: vi.fn(),
  };
  const requestingStates: boolean[] = [];
  const controller = new FullBackupRequestController(dependencies, (isRequesting) =>
    requestingStates.push(isRequesting),
  );

  return { controller, dependencies, requestingStates };
};

describe('FullBackupRequestController', () => {
  it('frees a busy button on the first landed refresh even when it never sent the request', () => {
    const { controller, requestingStates } = createRequestController(() => Promise.resolve());

    controller.handleRefreshLanded();

    expect(requestingStates).toEqual([false]);
  });

  it('refreshes after a successful request and keeps the button busy until that refresh lands', async () => {
    const { controller, dependencies, requestingStates } = createRequestController(() =>
      Promise.resolve(),
    );

    await controller.requestFullBackup();

    expect(dependencies.requestFullBackup).toHaveBeenCalledOnce();
    expect(dependencies.reportRequested).toHaveBeenCalledOnce();
    expect(dependencies.refreshNow).toHaveBeenCalledOnce();
    expect(requestingStates).toEqual([true]);

    controller.handleRefreshLanded();

    expect(requestingStates).toEqual([true, false]);
  });

  it('keeps the button busy when a scheduled refresh lands while the request is being sent', async () => {
    const deferredRequest = createDeferredRequest();
    const { controller, requestingStates } = createRequestController(() => deferredRequest.promise);

    const request = controller.requestFullBackup();
    controller.handleRefreshLanded();

    expect(requestingStates).toEqual([true]);

    deferredRequest.resolveRequest();
    await request;

    expect(requestingStates).toEqual([true]);
  });

  it('reports a failed request and frees the button at once', async () => {
    const refusal = new Error('refused');
    const { controller, dependencies, requestingStates } = createRequestController(() =>
      Promise.reject(refusal),
    );

    await controller.requestFullBackup();

    expect(dependencies.reportError).toHaveBeenCalledWith(refusal);
    expect(dependencies.reportRequested).not.toHaveBeenCalled();
    expect(dependencies.refreshNow).not.toHaveBeenCalled();
    expect(requestingStates).toEqual([true, false]);
  });

  it('ignores another press while the request is being sent', async () => {
    const deferredRequest = createDeferredRequest();
    const { controller, dependencies } = createRequestController(() => deferredRequest.promise);

    const firstRequest = controller.requestFullBackup();
    await controller.requestFullBackup();
    deferredRequest.resolveRequest();
    await firstRequest;

    expect(dependencies.requestFullBackup).toHaveBeenCalledOnce();
  });
});
