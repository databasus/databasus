// @vitest-environment happy-dom
import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';

import { RequestResetPasswordComponent } from '@/pages/auth/ui/RequestResetPasswordComponent';
import { buildProblem, mockApi, setUpMockApi } from '@/shared/lib/test/mockApi';
import { renderScreen } from '@/shared/lib/test/renderScreen';

const server = setUpMockApi();
const observeResponse = () =>
  new Promise<void>((resolve) => {
    const onResponse = () => {
      server.events.removeListener('response:mocked', onResponse);
      resolve();
    };
    server.events.on('response:mocked', onResponse);
  });

afterEach(() => vi.restoreAllMocks());
const submitRequest = () => {
  fireEvent.change(screen.getByRole('textbox', { name: 'Your email' }), {
    target: { value: 'owner@example.com' },
  });
  fireEvent.click(screen.getByRole('button', { name: 'Send reset code' }));
};

describe('RequestResetPasswordComponent', () => {
  it('prevents another delivery after success while navigation is pending', async () => {
    const onReset = vi.fn();
    const delivery = mockApi.delayed({
      method: 'post',
      path: '/password-resets',
      status: 204,
      body: undefined,
    });
    server.use(delivery.handler);

    renderScreen(<RequestResetPasswordComponent onSwitchToResetPassword={onReset} />);
    submitRequest();
    await waitFor(() => expect(server.getRequests()).toHaveLength(1));

    vi.useFakeTimers({ toFake: ['setTimeout', 'clearTimeout'] });
    const processed = observeResponse();
    await act(async () => {
      delivery.release();
      await processed;
    });

    expect(screen.getByText('If the email exists, a reset code has been sent')).toBeTruthy();

    const repeatedDelivery = mockApi.delayed({
      method: 'post',
      path: '/password-resets',
      status: 204,
      body: undefined,
    });
    server.use(repeatedDelivery.handler);

    const submit = screen.getByRole('button', { name: 'Send reset code' });
    expect(submit.hasAttribute('disabled')).toBe(true);
    const email = screen.getByRole('textbox', { name: 'Your email' });
    fireEvent.change(email, { target: { value: 'corrected@example.com' } });
    fireEvent.keyDown(email, { key: 'Enter', code: 'Enter', keyCode: 13 });
    fireEvent.click(submit);
    expect(screen.getByText('If the email exists, a reset code has been sent')).toBeTruthy();
    expect(server.getRequests()).toHaveLength(1);
    expect(onReset).not.toHaveBeenCalled();

    await act(async () => {
      await vi.advanceTimersByTimeAsync(1999);
    });
    expect(onReset).not.toHaveBeenCalled();

    await act(async () => {
      await vi.advanceTimersByTimeAsync(1);
    });

    expect(onReset).toHaveBeenCalledExactlyOnceWith('owner@example.com');
    expect(server.getRequests()).toHaveLength(1);
    repeatedDelivery.release();
  });

  it('opens completion for the submitted email after the code request succeeds', async () => {
    const onReset = vi.fn();
    vi.useFakeTimers({ toFake: ['setTimeout', 'clearTimeout'] });
    server.use(mockApi.post('/password-resets', 204, undefined));

    renderScreen(<RequestResetPasswordComponent onSwitchToResetPassword={onReset} />);
    const processed = observeResponse();
    await act(async () => {
      submitRequest();
      await processed;
    });

    expect(screen.getByText('If the email exists, a reset code has been sent')).toBeTruthy();

    await act(async () => {
      await vi.advanceTimersByTimeAsync(2000);
    });
    expect(onReset).toHaveBeenCalledExactlyOnceWith('owner@example.com');
  });

  it('retains the email with an explicit correction path when the server refuses delivery', async () => {
    server.use(mockApi.post('/password-resets', 403, buildProblem('permission_denied', 403)));
    renderScreen(<RequestResetPasswordComponent />);
    submitRequest();
    await screen.findByText('You do not have permission to do this.');
    expect(screen.getByDisplayValue('owner@example.com')).toBeTruthy();
    expect(screen.getByRole('button', { name: 'Send reset code' }).hasAttribute('disabled')).toBe(
      false,
    );
  });

  it('cancels delayed navigation when the successful request page is unmounted', async () => {
    const onReset = vi.fn();
    vi.useFakeTimers({ toFake: ['setTimeout', 'clearTimeout'] });
    server.use(mockApi.post('/password-resets', 204, undefined));
    const view = renderScreen(<RequestResetPasswordComponent onSwitchToResetPassword={onReset} />);

    const processed = observeResponse();
    await act(async () => {
      submitRequest();
      await processed;
    });

    expect(screen.getByText('If the email exists, a reset code has been sent')).toBeTruthy();
    view.rerender(<div>Reset request closed</div>);

    await act(async () => {
      await vi.advanceTimersByTimeAsync(1999);
    });
    expect(vi.getTimerCount()).toBe(0);

    await act(async () => {
      await vi.advanceTimersByTimeAsync(1);
    });
    expect(onReset).not.toHaveBeenCalled();
  });

  it('prevents repeated delivery and ignores a late response after navigation', async () => {
    const onReset = vi.fn();
    const request = mockApi.delayed({
      method: 'post',
      path: '/password-resets',
      status: 204,
      body: undefined,
    });
    server.use(request.handler);

    const view = renderScreen(<RequestResetPasswordComponent onSwitchToResetPassword={onReset} />);
    submitRequest();
    await waitFor(() => expect(server.getRequests()).toHaveLength(1));
    fireEvent.click(screen.getByRole('button', { name: /Send reset code/ }));
    expect(server.getRequests()).toHaveLength(1);

    view.rerender(<div>Reset request closed</div>);
    vi.useFakeTimers({ toFake: ['setTimeout', 'clearTimeout'] });
    const processed = observeResponse();
    await act(async () => {
      request.release();
      await processed;
    });

    await act(async () => {
      await vi.advanceTimersByTimeAsync(1999);
    });
    expect(vi.getTimerCount()).toBe(0);

    await act(async () => {
      await vi.advanceTimersByTimeAsync(1);
    });
    expect(onReset).not.toHaveBeenCalled();
    expect(screen.queryByText('If the email exists, a reset code has been sent')).toBeNull();
  });
});
