import { CloseCircleOutlined, EyeOutlined } from '@ant-design/icons';
import type { JSX } from 'react';
import { useTranslation } from 'react-i18next';

import type { Verification } from '@/pages/databases/model/verification/Verification';
import { VerificationStatus } from '@/pages/databases/model/verification/VerificationStatus';
import { RowActionButton } from '@/pages/databases/ui/RowActionButton';

export interface VerificationRowActions {
  isCanManageDBs: boolean;
  cancellingVerificationId?: string;
  onCancel: (verificationId: string) => void;
  onViewDetails: (verificationId: string) => void;
}

interface Props {
  verification: Verification;
  actions: VerificationRowActions;
}

export const VerificationActions = ({ verification, actions }: Props): JSX.Element => {
  const { t } = useTranslation();

  const { isCanManageDBs, cancellingVerificationId } = actions;

  const isUnfinished =
    verification.status === VerificationStatus.PENDING ||
    verification.status === VerificationStatus.RUNNING;

  if (isUnfinished && !isCanManageDBs) return <span className="text-xs text-muted">-</span>;

  return (
    <div className="flex gap-2">
      {isUnfinished ? (
        <RowActionButton
          title={t('verification.runs.actions.cancel')}
          icon={<CloseCircleOutlined />}
          className="!text-danger"
          isPending={cancellingVerificationId === verification.id}
          isDisabled={Boolean(cancellingVerificationId)}
          onClick={() => actions.onCancel(verification.id)}
        />
      ) : (
        <RowActionButton
          title={t('verification.runs.actions.viewDetails')}
          icon={<EyeOutlined />}
          className="!text-accent"
          onClick={() => actions.onViewDetails(verification.id)}
        />
      )}
    </div>
  );
};
