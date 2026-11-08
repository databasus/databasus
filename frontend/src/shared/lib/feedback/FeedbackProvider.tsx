import { useMantineEnv } from '@mantine/core';
import { createNotificationsStore, Notifications, notifications } from '@mantine/notifications';
import { CircleAlert, CircleCheck, Info, TriangleAlert } from 'lucide-react';
import { useCallback, useEffect, useMemo, useRef, useState, type ReactNode } from 'react';
import { useTranslation } from 'react-i18next';

import { subscribeSession } from '@/shared/auth';
import { useLatestRef } from '@/shared/hooks';
import { FeedbackContext } from '@/shared/lib/feedback/feedbackContext';
import type { FeedbackNotice } from '@/shared/lib/feedback/FeedbackNotice';

const noticePresentation = {
  success: { color: 'success', Icon: CircleCheck },
  error: { color: 'danger', Icon: CircleAlert },
  warning: { color: 'warning', Icon: TriangleAlert },
  info: { color: 'accent', Icon: Info },
};

export const FeedbackProvider = ({ children }: { children: ReactNode }) => {
  const [store] = useState(createNotificationsStore);
  const environment = useMantineEnv();
  const activeNotices = useRef(
    new Map<string, { owner?: string; timer?: ReturnType<typeof setTimeout> }>(),
  );
  const nextNoticeId = useRef(0);
  const { t } = useTranslation();
  const translation = useLatestRef(t);

  const dismiss = useCallback(
    (id: string) => {
      const notice = activeNotices.current.get(id);
      if (!notice) return;

      if (notice?.timer !== undefined) clearTimeout(notice.timer);
      activeNotices.current.delete(id);
      notifications.hide(id, store);
    },
    [store],
  );

  const dismissOwner = useCallback(
    (owner: string) => {
      for (const [id, notice] of activeNotices.current) if (notice.owner === owner) dismiss(id);
    },
    [dismiss],
  );

  const show = useCallback(
    (notice: FeedbackNotice, owner?: string) => {
      if (owner) dismissOwner(owner);
      // eslint-disable-next-line i18next/no-literal-string -- Notification store identifier
      const id = `feedback-${++nextNoticeId.current}`;
      const { color, Icon } = noticePresentation[notice.kind];
      const lifetimeMs = notice.lifetimeMs ?? 3000;
      const timer = lifetimeMs > 0 ? setTimeout(() => dismiss(id), lifetimeMs) : undefined;
      activeNotices.current.set(id, { owner, timer });

      notifications.show(
        {
          id,
          title: notice.description === undefined ? undefined : notice.title,
          message: notice.description ?? notice.title,
          color,
          icon: <Icon aria-hidden size={16} />,
          position: notice.position ?? 'top-center',
          autoClose: false,
          withCloseButton: notice.description !== undefined,
          closeButtonProps: { 'aria-label': translation.current('common.actions.close') },
          onClose: () => dismiss(id),
          role: notice.kind === 'error' ? 'alert' : 'status',
        },
        store,
      );
    },
    [dismiss, dismissOwner, store, translation],
  );

  useEffect(() => {
    const clearNotices = () => {
      for (const id of activeNotices.current.keys()) dismiss(id);
    };
    const unsubscribe = subscribeSession(clearNotices);

    return () => {
      unsubscribe();
      clearNotices();
    };
  }, [dismiss]);

  const value = useMemo(() => ({ show, dismissOwner }), [show, dismissOwner]);

  return (
    <FeedbackContext.Provider value={value}>
      {children}
      <Notifications
        store={store}
        autoClose={false}
        transitionDuration={environment === 'test' ? 0 : undefined}
      />
    </FeedbackContext.Provider>
  );
};
