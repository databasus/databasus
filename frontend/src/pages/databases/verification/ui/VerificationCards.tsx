import { Loader } from '@mantine/core';
import dayjs from 'dayjs';
import type { JSX } from 'react';
import { useTranslation } from 'react-i18next';

import type { Verification } from '@/pages/databases/verification/model/verificationContract';
import { VerificationStatus } from '@/pages/databases/verification/model/VerificationStatus';
import {
  VerificationActions,
  type VerificationRowActions,
} from '@/pages/databases/verification/ui/VerificationActions';
import { VerificationStatusLabel } from '@/pages/databases/verification/ui/VerificationStatusLabel';
import { VerificationTriggerTag } from '@/pages/databases/verification/ui/VerificationTriggerTag';
import { useLocale } from '@/shared/i18n';
import { getUserTimeFormat } from '@/shared/time';

interface Props {
  verifications: Verification[];
  isLoading: boolean;
  actions: VerificationRowActions;
}

export const VerificationCards = ({ verifications, isLoading, actions }: Props): JSX.Element => {
  const { t } = useTranslation();
  const { formatRelativeTime } = useLocale();

  if (isLoading) {
    return (
      <div className="flex justify-center py-8">
        <Loader role="status" />
      </div>
    );
  }

  return (
    <div>
      {verifications.map((verification) => (
        <div
          key={verification.id}
          className="mb-2 rounded-lg border border-border bg-surface p-4 shadow-sm"
        >
          <div className="flex items-start justify-between">
            <div>
              <div className="text-xs text-muted">{t('verification.runs.fields.createdAt')}</div>
              <div className="text-sm font-medium">
                {dayjs.utc(verification.createdAt).local().format(getUserTimeFormat().format)}
              </div>
              <div className="text-xs text-muted">
                ({formatRelativeTime(dayjs.utc(verification.createdAt))})
              </div>
            </div>
            <VerificationStatusLabel status={verification.status} />
          </div>

          <div className="mt-3 flex items-center gap-2">
            <VerificationTriggerTag trigger={verification.trigger} />
            {verification.attemptCount > 1 && (
              <span className="text-xs text-muted">
                {t('verification.runs.attempt', { attempt: verification.attemptCount })}
              </span>
            )}
          </div>

          {verification.failMessage && (
            <div
              className={`mt-2 text-xs ${verification.status === VerificationStatus.CANCELED ? 'text-muted' : 'text-danger'}`}
            >
              {verification.failMessage}
            </div>
          )}

          <div className="mt-3 flex items-center justify-end border-t border-border pt-3">
            <VerificationActions verification={verification} actions={actions} />
          </div>
        </div>
      ))}
    </div>
  );
};
