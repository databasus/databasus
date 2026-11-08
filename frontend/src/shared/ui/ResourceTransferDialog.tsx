import { Alert, Button, Loader } from '@mantine/core';
import { CircleAlert } from 'lucide-react';
import { useEffect, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { useSessionMutation } from '@/shared/api';
import { useSessionQuery } from '@/shared/api';
import { useSessionGuard } from '@/shared/auth';
import { useLatestRef, useRequestRevision } from '@/shared/hooks';
import { translateApiError } from '@/shared/i18n';
import { Dialog } from '@/shared/ui/Dialog';
import { LoadFailure } from '@/shared/ui/LoadFailure';
import { WrappingSelect } from '@/shared/ui/WrappingSelect';
export type TransferAvailability =
  | {
      isBlocked: true;
    }
  | {
      isBlocked: false;
      targets: {
        label: string;
        value: string;
      }[];
    };
interface Props {
  contextKey: string;
  loadAvailability: (signal: AbortSignal) => Promise<TransferAvailability | undefined>;
  transfer: (targetId: string) => Promise<void>;
  onClose: () => void;
  onTransferred: () => void;
  labels: {
    title: string;
    blocked: string;
    description: string;
    target: string;
    placeholder: string;
    submit: string;
  };
}

export const ResourceTransferDialog = (props: Props) => {
  const { contextKey, labels } = props;
  const { t } = useTranslation();
  const query = useSessionQuery({
    queryKey: ['resource-transfer', contextKey],
    queryFn: (signal) => props.loadAvailability(signal),
  });
  const availability = query.data;
  const loadError = query.error ?? undefined;
  const [transferError, setTransferError] = useState<unknown>();
  const isLoading = query.isPending;
  const [selectedTargetId, setSelectedTargetId] = useState<string>();
  const transfer = useSessionMutation({
    mutationFn: (targetId: string) => props.transfer(targetId),
  });
  const isTransferring = transfer.isPending;
  const { reset: resetTransfer } = transfer;
  const [invokingControl] = useState(() =>
    document.activeElement instanceof HTMLElement ? document.activeElement : null,
  );
  const isMounted = useRef(false);
  const hasPendingTransfer = useRef(false);
  const isCurrent = useSessionGuard();
  const revision = useRequestRevision();
  const latestProps = useLatestRef(props);

  useEffect(() => {
    isMounted.current = true;

    return () => {
      isMounted.current = false;
      queueMicrotask(() => {
        if (!isMounted.current && invokingControl?.isConnected) invokingControl.focus();
      });
    };
  }, [invokingControl]);

  const loadData = () => {
    setTransferError(undefined);
    return query.refetch();
  };

  const transferResource = async () => {
    if (!isCurrent() || !selectedTargetId || hasPendingTransfer.current) return;

    const requestRevision = revision.GetCurrent();
    const isActive = () => isCurrent() && revision.IsCurrent(requestRevision);

    hasPendingTransfer.current = true;
    setTransferError(undefined);

    try {
      await transfer.mutateAsync(selectedTargetId);
      if (isActive()) latestProps.current.onTransferred();
    } catch (error) {
      if (isActive()) setTransferError(error);
    } finally {
      if (isActive()) {
        hasPendingTransfer.current = false;
        resetTransfer();
      }
    }
  };

  useEffect(() => {
    setSelectedTargetId(undefined);
    resetTransfer();
    hasPendingTransfer.current = false;

    return () => {
      revision.Invalidate();
    };
  }, [contextKey, revision, resetTransfer]);

  const close = () => {
    revision.Invalidate();
    props.onClose();
  };

  return (
    <Dialog title={labels.title} opened onClose={close} closeOnClickOutside={false}>
      {loadError !== undefined && <LoadFailure error={loadError} onRetry={() => void loadData()} />}
      {transferError !== undefined && (
        <Alert
          color="danger"
          icon={
            <CircleAlert aria-hidden size={16} className="inline-block shrink-0 align-middle" />
          }
          title={translateApiError(transferError, t)}
        />
      )}
      {isLoading ? (
        <div className="flex justify-center py-5">
          <Loader role="status" />
        </div>
      ) : availability?.isBlocked ? (
        <div className="py-3">
          <div className="text-muted">{labels.blocked}</div>
          <div className="mt-5">
            <Button
              variant="filled"
              className="!h-auto min-h-8 max-w-full !whitespace-normal"
              onClick={close}
            >
              {t('common.actions.ok')}
            </Button>
          </div>
        </div>
      ) : availability ? (
        <div className="py-3">
          <div className="mb-3 text-muted">{labels.description}</div>
          <div className="mb-5 flex min-w-0 flex-col gap-2 sm:flex-row sm:items-center">
            <div className="min-w-30 pr-2">{labels.target}</div>
            <WrappingSelect
              aria-label={labels.target}
              value={selectedTargetId}
              onChange={setSelectedTargetId}
              className="min-w-0 grow"
              placeholder={labels.placeholder}
              data={availability.targets}
            />
          </div>
          <div className="flex flex-wrap gap-2">
            <Button
              variant="default"
              className="!h-auto min-h-8 max-w-full !whitespace-normal"
              onClick={close}
            >
              {t('common.actions.cancel')}
            </Button>
            <Button
              variant="filled"
              className="!h-auto min-h-8 max-w-full !whitespace-normal"
              onClick={() => void transferResource()}
              loading={isTransferring}
              disabled={!selectedTargetId || isTransferring}
              aria-busy={isTransferring}
            >
              {labels.submit}
            </Button>
          </div>
        </div>
      ) : null}
    </Dialog>
  );
};
