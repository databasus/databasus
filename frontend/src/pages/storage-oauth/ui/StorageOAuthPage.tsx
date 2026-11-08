import { Alert, Loader } from '@mantine/core';
import { Info } from 'lucide-react';
import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';

import {
  getStorageAuthorizationChannelName,
  isStorageAuthorizationState,
  type StorageAuthorizationResponse,
} from '@/features/edit-storage';

const getCallbackResponse = (): StorageAuthorizationResponse | undefined => {
  const parameters = new URLSearchParams(window.location.search);
  const state = parameters.get('state');
  const code = parameters.get('code');
  const error = parameters.get('error');

  if (
    !isStorageAuthorizationState(state) ||
    parameters.getAll('state').length !== 1 ||
    parameters.getAll('code').length > 1 ||
    parameters.getAll('error').length > 1
  )
    return;
  if (code?.trim() && !parameters.has('error')) return { type: 'code', state, code };
  if (error?.trim() && !parameters.has('code')) return { type: 'refusal', state };
};

export function StorageOAuthPage() {
  const [response] = useState(getCallbackResponse);
  const [status, setStatus] = useState<'waiting' | 'complete' | 'restart'>(
    response ? 'waiting' : 'restart',
  );
  const { t } = useTranslation();

  useEffect(() => {
    window.history.replaceState(window.history.state, '', window.location.pathname);
    if (!response) return;

    let channel: BroadcastChannel;
    let timer: ReturnType<typeof setTimeout>;
    const release = () => {
      clearTimeout(timer);
      channel?.close();
    };
    try {
      channel = new BroadcastChannel(getStorageAuthorizationChannelName(response.state));
      channel.onmessage = (event: MessageEvent<unknown>) => {
        const message = event.data;
        if (
          event.origin !== window.location.origin ||
          !message ||
          typeof message !== 'object' ||
          Object.keys(message).sort().join(',') !== 'state,type' ||
          !('type' in message) ||
          message.type !== 'ack' ||
          !('state' in message) ||
          message.state !== response.state
        )
          return;
        release();
        setStatus('complete');
        window.close();
      };
      timer = setTimeout(() => {
        release();
        setStatus('restart');
      }, 5000);
      channel.postMessage(response);
    } catch {
      release();
      setStatus('restart');
    }
    return release;
  }, [response]);

  return (
    <div className="mx-auto w-full max-w-lg p-4">
      {status === 'waiting' ? (
        <div role="status">
          <Loader aria-hidden />
          {t('app.oauthStorage.waiting')}
        </div>
      ) : (
        <Alert
          color={status === 'complete' ? 'success' : 'danger'}
          icon={<Info aria-hidden size={16} className="inline-block shrink-0 align-middle" />}
          title={t(
            status === 'complete' ? 'app.oauthStorage.complete' : 'app.oauthStorage.restart',
          )}
        />
      )}
    </div>
  );
}
