import {
  CheckCircleOutlined,
  ClockCircleOutlined,
  CloseCircleOutlined,
  ExclamationCircleOutlined,
  SyncOutlined,
} from '@ant-design/icons';
import type { JSX, ReactNode } from 'react';
import { useTranslation } from 'react-i18next';

import { VerificationStatus } from '../../model/verification/VerificationStatus';
import { VERIFICATION_STATUS_LABEL_KEYS } from '../../model/verification/VerificationStatusLabelKeys';

interface Props {
  status: VerificationStatus;
}

const STATUS_PRESENTATION: Record<VerificationStatus, { icon: ReactNode; className: string }> = {
  [VerificationStatus.COMPLETED]: { icon: <CheckCircleOutlined />, className: 'text-success' },
  [VerificationStatus.FAILED]: { icon: <ExclamationCircleOutlined />, className: 'text-danger' },
  [VerificationStatus.RUNNING]: { icon: <SyncOutlined spin />, className: 'font-bold text-accent' },
  [VerificationStatus.PENDING]: { icon: <ClockCircleOutlined />, className: 'text-muted' },
  [VerificationStatus.CANCELED]: { icon: <CloseCircleOutlined />, className: 'text-muted' },
};

export const VerificationStatusLabel = ({ status }: Props): JSX.Element => {
  const { t } = useTranslation();

  const { icon, className } = STATUS_PRESENTATION[status];

  return (
    <div className={`flex items-center gap-2 ${className}`}>
      {icon}
      <span>{t(VERIFICATION_STATUS_LABEL_KEYS[status])}</span>
    </div>
  );
};
