import { Loader, Tooltip } from '@mantine/core';
import dayjs from 'dayjs';
import { CircleAlert } from 'lucide-react';
import type { JSX } from 'react';
import { useTranslation } from 'react-i18next';

import type { Verification } from '@/pages/databases/verification/model/verificationContract';
import { formatOptionalDuration } from '@/pages/databases/verification/ui/formatOptionalDuration';
import {
  VerificationActions,
  type VerificationRowActions,
} from '@/pages/databases/verification/ui/VerificationActions';
import { VerificationStatusLabel } from '@/pages/databases/verification/ui/VerificationStatusLabel';
import { VerificationTriggerTag } from '@/pages/databases/verification/ui/VerificationTriggerTag';
import { useLocale } from '@/shared/i18n';
import { getUserTimeFormat } from '@/shared/time';
import tableStyles from '@/shared/ui/nativeTable.module.css';

interface Props {
  verifications: Verification[];
  isLoading: boolean;
  actions: VerificationRowActions;
}

export const VerificationsTable = ({ verifications, isLoading, actions }: Props): JSX.Element => {
  const { t } = useTranslation();
  const { formatRelativeTime } = useLocale();

  return (
    <div className="relative overflow-x-auto" aria-busy={isLoading}>
      <table className={`${tableStyles.table} ${tableStyles.bordered}`}>
        <thead>
          <tr>
            <th scope="col" style={{ width: 220 }}>
              {t('verification.runs.fields.createdAt')}
            </th>
            <th scope="col" style={{ width: 160 }}>
              {t('common.fields.status')}
            </th>
            <th scope="col" style={{ width: 110 }}>
              {t('verification.runs.fields.trigger')}
            </th>
            <th scope="col" style={{ width: 130 }}>
              <div className="flex items-center">
                {t('verification.runs.fields.duration')}
                <Tooltip className="ml-1" label={t('verification.runs.durationTooltip')}>
                  <CircleAlert size="1em" className="inline-block shrink-0 align-middle" />
                </Tooltip>
              </div>
            </th>
            <th scope="col" style={{ width: 120 }}>
              {t('verification.runs.fields.actions')}
            </th>
          </tr>
        </thead>
        <tbody>
          {verifications.map((verification) => (
            <tr key={verification.id}>
              <td>
                <div>
                  {dayjs.utc(verification.createdAt).local().format(getUserTimeFormat().format)}{' '}
                  <br />
                  <span className="text-muted">
                    ({formatRelativeTime(dayjs.utc(verification.createdAt))})
                  </span>
                </div>
              </td>
              <td>
                <VerificationStatusLabel status={verification.status} />
              </td>
              <td>
                <VerificationTriggerTag trigger={verification.trigger} />
              </td>
              <td>
                {(() => {
                  const { restoreDurationMs, verifyDurationMs } = verification;

                  if (restoreDurationMs === undefined && verifyDurationMs === undefined) return '-';

                  const totalMs = (restoreDurationMs ?? 0) + (verifyDurationMs ?? 0);

                  return (
                    <div className="text-sm">
                      <div>{formatOptionalDuration(totalMs)}</div>
                      <div className="text-xs text-muted">
                        {t('verification.runs.restoreDuration', {
                          duration: formatOptionalDuration(restoreDurationMs),
                        })}
                      </div>
                    </div>
                  );
                })()}
              </td>
              <td>
                <VerificationActions verification={verification} actions={actions} />
              </td>
            </tr>
          ))}
          {verifications.length === 0 && !isLoading && (
            <tr>
              <td colSpan={5}>
                <div className="py-8 text-center text-muted">{t('verification.runs.empty')}</div>
              </td>
            </tr>
          )}
        </tbody>
      </table>
      {isLoading && (
        <div className="absolute inset-0 flex items-center justify-center bg-surface/50">
          <Loader size="sm" role="status" />
        </div>
      )}
    </div>
  );
};
