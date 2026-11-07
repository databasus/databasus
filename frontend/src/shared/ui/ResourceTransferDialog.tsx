import { Alert, Button, Modal, Select, Spin } from 'antd';
import { useEffect, useEffectEvent, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { useSessionGuard } from '@/shared/auth';
import { useLatestRef, useRequestRevision } from '@/shared/hooks';
import { translateApiError } from '@/shared/i18n';
import { LoadFailure } from '@/shared/ui/LoadFailure';
import { WRAPPING_SELECT_CLASS_NAME } from '@/shared/ui/wrappingSelectClassName';

export type TransferAvailability =
  | { isBlocked: true }
  | { isBlocked: false; targets: { label: string; value: string }[] };

interface Props {
  contextKey: string;
  loadAvailability: (isActive: () => boolean) => Promise<TransferAvailability | undefined>;
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
  const [availability, setAvailability] = useState<TransferAvailability>();
  const [loadError, setLoadError] = useState<unknown>();
  const [transferError, setTransferError] = useState<unknown>();
  const [isLoading, setIsLoading] = useState(true);
  const [selectedTargetId, setSelectedTargetId] = useState<string>();
  const [isTransferring, setIsTransferring] = useState(false);
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

  const loadData = async () => {
    if (!isCurrent()) return;

    const requestRevision = revision.Invalidate();
    const isActive = () => isCurrent() && revision.IsCurrent(requestRevision);

    setLoadError(undefined);
    setTransferError(undefined);
    setIsLoading(true);

    try {
      const loadedAvailability = await latestProps.current.loadAvailability(isActive);
      if (isActive()) setAvailability(loadedAvailability);
    } catch (error) {
      if (isActive()) setLoadError(error);
    } finally {
      if (isActive()) setIsLoading(false);
    }
  };

  const transferResource = async () => {
    if (!isCurrent() || !selectedTargetId || hasPendingTransfer.current) return;

    const requestRevision = revision.GetCurrent();
    const isActive = () => isCurrent() && revision.IsCurrent(requestRevision);

    hasPendingTransfer.current = true;
    setIsTransferring(true);
    setTransferError(undefined);

    try {
      await latestProps.current.transfer(selectedTargetId);
      if (isActive()) latestProps.current.onTransferred();
    } catch (error) {
      if (isActive()) setTransferError(error);
    } finally {
      if (isActive()) {
        hasPendingTransfer.current = false;
        setIsTransferring(false);
      }
    }
  };

  const startDataLoad = useEffectEvent(() => {
    setAvailability(undefined);
    setSelectedTargetId(undefined);
    setIsTransferring(false);
    hasPendingTransfer.current = false;

    void loadData();
  });

  useEffect(() => {
    startDataLoad();

    return () => {
      revision.Invalidate();
    };
  }, [contextKey, revision]);

  const close = () => {
    revision.Invalidate();
    props.onClose();
  };

  return (
    <Modal title={labels.title} footer={null} open onCancel={close} maskClosable={false}>
      {loadError !== undefined && <LoadFailure error={loadError} onRetry={() => void loadData()} />}
      {transferError !== undefined && (
        <Alert type="error" showIcon message={translateApiError(transferError, t)} />
      )}
      {isLoading ? (
        <div className="flex justify-center py-5">
          <Spin />
        </div>
      ) : availability?.isBlocked ? (
        <div className="py-3">
          <div className="text-muted">{labels.blocked}</div>
          <div className="mt-5">
            <Button
              type="primary"
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
            <Select
              aria-label={labels.target}
              value={selectedTargetId}
              onChange={setSelectedTargetId}
              className={`${WRAPPING_SELECT_CLASS_NAME} min-w-0 grow [&_.ant-select-selection-placeholder]:!relative [&_.ant-select-selection-placeholder]:!inset-x-0 [&_.ant-select-selection-placeholder]:!top-auto [&_.ant-select-selection-placeholder]:!transform-none [&_.ant-select-selection-placeholder]:!leading-normal [&_.ant-select-selection-placeholder]:!whitespace-normal`}
              placeholder={labels.placeholder}
              options={availability.targets}
            />
          </div>
          <div className="flex flex-wrap gap-2">
            <Button
              type="default"
              className="!h-auto min-h-8 max-w-full !whitespace-normal"
              onClick={close}
            >
              {t('common.actions.cancel')}
            </Button>
            <Button
              type="primary"
              className="!h-auto min-h-8 max-w-full !whitespace-normal"
              onClick={() => void transferResource()}
              loading={isTransferring}
              disabled={!selectedTargetId || isTransferring}
            >
              {labels.submit}
            </Button>
          </div>
        </div>
      ) : null}
    </Modal>
  );
};
