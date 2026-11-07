import { Alert, Loader } from '@mantine/core';
import { TriangleAlert } from 'lucide-react';
import { useEffect, useState } from 'react';
import { Trans, useTranslation } from 'react-i18next';

import type { Database } from '@/entities/databases';
import { verificationAgentApi } from '@/entities/verification-agents';
import { verificationRunsApi } from '@/pages/databases/api/verificationRunsApi';
import { type Verification } from '@/pages/databases/model/verification/Verification';
import type { VerificationRowActions } from '@/pages/databases/ui/verification/VerificationActions';
import { VerificationCards } from '@/pages/databases/ui/verification/VerificationCards';
import { VerificationDetailDrawer } from '@/pages/databases/ui/verification/VerificationDetailDrawer';
import { VerificationsTable } from '@/pages/databases/ui/verification/VerificationsTable';
import { useSessionGuard } from '@/shared/auth';
import { usePolledPagedList } from '@/shared/hooks';
import { getWebsitePageUrl, translateApiError, useLocale } from '@/shared/i18n';
import { useFeedback } from '@/shared/lib/feedback';
import { LoadFailure } from '@/shared/ui';

interface Props {
  database: Database;
  isCanManageDBs: boolean;
  scrollContainerRef?: React.RefObject<HTMLDivElement | null>;
}
const AGENT_AVAILABILITY_REFRESH_MS = 5000;
const VERIFICATIONS_REFRESH_MS = 1000;
const VERIFICATIONS_PAGE_SIZE = 50;

export const VerificationsComponent = ({ database, isCanManageDBs, scrollContainerRef }: Props) => {
  const { t } = useTranslation();
  const { locale } = useLocale();
  const notify = useFeedback();
  const isActive = useSessionGuard();

  const [detailVerificationId, setDetailVerificationId] = useState<string | undefined>();
  const [isDetailOpen, setIsDetailOpen] = useState(false);
  const [hasAgents, setHasAgents] = useState<boolean | undefined>();
  const [cancellingVerificationId, setCancellingVerificationId] = useState<string | undefined>();

  const fetchVerificationsPage = async (after: string | undefined) => {
    const response = await verificationRunsApi.listByDatabase(
      database.id,
      VERIFICATIONS_PAGE_SIZE,
      after,
    );
    return { items: response.items, nextCursor: response.nextCursor };
  };

  const cancelVerification = async (verificationId: string) => {
    if (cancellingVerificationId) return;

    setCancellingVerificationId(verificationId);

    try {
      await verificationRunsApi.cancel(verificationId);
      if (isActive()) await refreshVerifications();
    } catch (error) {
      if (isActive()) notify({ kind: 'error', title: translateApiError(error, t) });
    }

    setCancellingVerificationId(undefined);
  };

  const openDetail = (verificationId: string) => {
    setDetailVerificationId(verificationId);
    setIsDetailOpen(true);
  };

  const {
    items: verifications,
    isLoading,
    isLoadingMore,
    hasMore,
    firstLoadError,
    reload: reloadVerifications,
    refreshFirstPage: refreshVerifications,
  } = usePolledPagedList({
    fetchPage: fetchVerificationsPage,
    getItemKey: (verification: Verification) => verification.id,
    refreshIntervalMs: VERIFICATIONS_REFRESH_MS,
    scrollContainerRef,
    resetKey: database.id,
    onError: (error, failedPage) => {
      if (failedPage === 'next') notify({ kind: 'error', title: translateApiError(error, t) });
    },
  });

  useEffect(() => {
    let isCancelled = false;

    const loadAvailability = () => {
      verificationAgentApi
        .getAgentCount()
        .then((availability) => {
          if (isCancelled) return;
          setHasAgents(availability.agentCount > 0);
        })
        .catch(() => {
          if (isCancelled) return;
          setHasAgents(undefined);
        });
    };

    loadAvailability();
    const intervalId = setInterval(loadAvailability, AGENT_AVAILABILITY_REFRESH_MS);

    return () => {
      isCancelled = true;
      clearInterval(intervalId);
    };
  }, []);

  const rowActions: VerificationRowActions = {
    isCanManageDBs,
    cancellingVerificationId,
    onCancel: (verificationId) => void cancelVerification(verificationId),
    onViewDetails: openDetail,
  };

  return (
    <div className="w-full rounded-md bg-surface p-3 shadow md:p-5">
      <h2 className="text-lg font-bold md:text-xl">{t('verification.runs.title')}</h2>

      <p className="mt-2 max-w-2xl text-sm text-muted">{t('verification.runs.description')}</p>

      {hasAgents === false && (
        <Alert
          className="!mt-3 max-w-[400px]"
          color="warning"
          icon={
            <TriangleAlert aria-hidden size={16} className="inline-block shrink-0 align-middle" />
          }
          title={
            <Trans
              i18nKey="verification.runs.noAgentsWarning"
              components={{
                docsLink: (
                  <a
                    href={getWebsitePageUrl('restoreVerification', locale)}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="underline hover:no-underline"
                  />
                ),
              }}
            />
          }
        />
      )}

      <div className="mt-5 w-full md:max-w-[900px]">
        {firstLoadError !== undefined ? (
          <LoadFailure error={firstLoadError} onRetry={() => void reloadVerifications()} />
        ) : (
          <>
            <div className="hidden md:block">
              <VerificationsTable
                verifications={verifications}
                isLoading={isLoading}
                actions={rowActions}
              />
              {!isLoading && verifications.length === 0 && (
                <div className="mt-3 text-center text-sm text-muted">
                  {t('verification.runs.empty')}
                </div>
              )}
              {isLoadingMore && (
                <div className="mt-2 flex justify-center">
                  <Loader role="status" />
                </div>
              )}
              {!hasMore && verifications.length > 0 && (
                <div className="mt-2 text-center text-muted">
                  {t('verification.runs.allLoaded', { total: verifications.length })}
                </div>
              )}
            </div>

            <div className="md:hidden">
              <VerificationCards
                verifications={verifications}
                isLoading={isLoading}
                actions={rowActions}
              />
              {!isLoading && verifications.length === 0 && (
                <div className="py-8 text-center text-muted">{t('verification.runs.empty')}</div>
              )}
              {isLoadingMore && (
                <div className="mt-3 flex justify-center">
                  <Loader role="status" />
                </div>
              )}
              {!hasMore && verifications.length > 0 && (
                <div className="mt-3 text-center text-sm text-muted">
                  {t('verification.runs.allLoaded', { total: verifications.length })}
                </div>
              )}
            </div>
          </>
        )}
      </div>

      {detailVerificationId && (
        <VerificationDetailDrawer
          verificationId={detailVerificationId}
          isOpen={isDetailOpen}
          onClose={() => setIsDetailOpen(false)}
          onClosed={() => setDetailVerificationId(undefined)}
        />
      )}
    </div>
  );
};
