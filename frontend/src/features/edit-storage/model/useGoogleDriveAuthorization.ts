import { useEffect, useRef, useState } from 'react';

import type { GoogleDriveStorageSettings } from '@/entities/storages';
import type { StorageAuthorizationAttempt } from '@/features/edit-storage/model/StorageAuthorizationAttempt';
import {
  getStorageAuthorizationChannelName,
  isStorageAuthorizationResponse,
} from '@/features/edit-storage/model/storageAuthorizationRelay';
import { getSessionSnapshot, isCurrentSession, subscribeSession } from '@/shared/auth';
import { useLatestRef } from '@/shared/hooks';
import type { TranslationKey } from '@/shared/i18n';

interface GoogleDriveAuthorizationOptions {
  workspaceId: string;
  storageId?: string;
  settings: GoogleDriveStorageSettings;
  onChange: (settings: GoogleDriveStorageSettings) => void;
}

const ATTEMPT_LIFETIME = 10 * 60 * 1000;

const getCurrentTimestamp = () => Date.now();

export function useGoogleDriveAuthorization(options: GoogleDriveAuthorizationOptions) {
  const [status, setStatus] = useState<'idle' | 'waiting' | 'exchanging' | 'authorized' | 'error'>(
    'idle',
  );
  const [errorKey, setErrorKey] = useState<TranslationKey>();

  const release = () => {
    const attempt = activeAttempt.current;
    activeAttempt.current = undefined;
    requestRevision.current++;

    if (!attempt) return;

    clearTimeout(attempt.timer);
    attempt.channel.close();
  };

  const cancel = () => {
    release();
    setStatus('idle');
    setErrorKey(undefined);
  };

  const fail = (key: TranslationKey) => {
    release();
    setErrorKey(key);
    setStatus('error');
  };

  const isCurrentAttempt = (attempt: StorageAuthorizationAttempt) => {
    const current = latestOptions.current;

    return (
      activeAttempt.current === attempt &&
      attempt.revision === requestRevision.current &&
      attempt.editorId === editorId.current &&
      isCurrentSession(attempt.session) &&
      current.workspaceId === attempt.workspaceId &&
      current.storageId === attempt.storageId &&
      current.settings.clientId === attempt.settings.clientId &&
      current.settings.clientSecret === attempt.settings.clientSecret
    );
  };

  const exchange = async (attempt: StorageAuthorizationAttempt, code: string) => {
    setStatus('exchanging');

    try {
      const response = await fetch('https://oauth2.googleapis.com/token', {
        method: 'POST',
        credentials: 'omit',
        redirect: 'error',
        headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
        body: new URLSearchParams({
          code,
          client_id: attempt.settings.clientId,
          client_secret: attempt.settings.clientSecret!,
          redirect_uri: attempt.redirectUri,
          grant_type: 'authorization_code',
        }),
      });
      if (!isCurrentAttempt(attempt)) return;

      if (!response.ok) {
        fail('storages.edit.googleDrive.exchangeFailed');
        return;
      }
      const token: unknown = await response.json();
      if (!isCurrentAttempt(attempt)) return;

      if (
        !token ||
        typeof token !== 'object' ||
        !('access_token' in token) ||
        typeof token.access_token !== 'string' ||
        !token.access_token.trim()
      ) {
        fail('storages.edit.googleDrive.exchangeFailed');
        return;
      }

      if (
        !('refresh_token' in token) ||
        typeof token.refresh_token !== 'string' ||
        !token.refresh_token.trim()
      ) {
        fail('storages.edit.googleDrive.missingRefreshToken');
        return;
      }

      latestOptions.current.onChange({
        ...latestOptions.current.settings,
        tokenJson: JSON.stringify(token),
      });
      release();
      setStatus('authorized');
    } catch {
      if (isCurrentAttempt(attempt)) fail('storages.edit.googleDrive.exchangeFailed');
    }
  };

  const start = () => {
    cancel();
    const current = latestOptions.current;

    if (!current.settings.clientId || !current.settings.clientSecret?.trim()) return;

    try {
      const state = Array.from(crypto.getRandomValues(new Uint8Array(32)), (byte) =>
        byte.toString(16).padStart(2, '0'),
      ).join('');
      const channel = new BroadcastChannel(getStorageAuthorizationChannelName(state));
      const attempt: StorageAuthorizationAttempt = {
        state,
        channel,
        createdAt: getCurrentTimestamp(),
        redirectUri: `${window.location.origin}/storages/google-oauth`,
        workspaceId: current.workspaceId,
        storageId: current.storageId,
        editorId: editorId.current,
        session: getSessionSnapshot(),
        settings: { ...current.settings },
        revision: requestRevision.current,
        isConsumed: false,
        timer: setTimeout(() => {
          if (isCurrentAttempt(attempt)) fail('storages.edit.googleDrive.expired');
        }, ATTEMPT_LIFETIME),
      };

      activeAttempt.current = attempt;
      channel.onmessage = (event: MessageEvent<unknown>) => {
        if (
          event.origin !== window.location.origin ||
          !isStorageAuthorizationResponse(event.data) ||
          event.data.state !== state ||
          !isCurrentAttempt(attempt) ||
          attempt.isConsumed
        )
          return;
        const age = Date.now() - attempt.createdAt;

        if (age < 0 || age >= ATTEMPT_LIFETIME) {
          fail('storages.edit.googleDrive.expired');
          return;
        }
        attempt.isConsumed = true;
        channel.postMessage({ type: 'ack', state });

        if (event.data.type === 'refusal') fail('storages.edit.googleDrive.refused');
        else void exchange(attempt, event.data.code);
      };

      const authorizationUrl = new URL('https://accounts.google.com/o/oauth2/v2/auth');
      authorizationUrl.search = new URLSearchParams({
        client_id: current.settings.clientId,
        redirect_uri: attempt.redirectUri,
        response_type: 'code',
        scope: 'https://www.googleapis.com/auth/drive.file',
        access_type: 'offline',
        prompt: 'consent',
        state,
      }).toString();
      setStatus('waiting');

      // eslint-disable-next-line i18next/no-literal-string -- browser window target
      if (!window.open(authorizationUrl.href, '_blank'))
        fail('storages.edit.googleDrive.popupBlocked');
    } catch {
      fail('storages.edit.googleDrive.unavailable');
    }
  };

  const activeAttempt = useRef<StorageAuthorizationAttempt | undefined>(undefined);
  const requestRevision = useRef(0);
  const editorId = useRef(Symbol());
  const latestOptions = useLatestRef(options);

  useEffect(() => {
    cancel();
    const unsubscribe = subscribeSession(cancel);

    return () => {
      unsubscribe();
      release();
    };
  }, [
    options.workspaceId,
    options.storageId,
    options.settings.clientId,
    options.settings.clientSecret,
  ]);

  return {
    start,
    cancel,
    status,
    errorKey,
    isPending: status === 'waiting' || status === 'exchanging',
  };
}
