import { CircleAlert, CircleCheck, CircleX, Clock3, RefreshCw } from 'lucide-react';
import type { JSX, ReactNode } from 'react';
import { useTranslation } from 'react-i18next';

import {
  VerificationStatus,
  VERIFICATION_STATUS_LABEL_KEYS,
} from '@/pages/databases/verification/model/VerificationStatus';

interface Props {
  status: VerificationStatus;
}
const STATUS_PRESENTATION: Record<
  VerificationStatus,
  {
    icon: ReactNode;
    className: string;
  }
> = {
  [VerificationStatus.COMPLETED]: { icon: <CircleCheck size={14} />, className: 'text-success' },
  [VerificationStatus.FAILED]: { icon: <CircleAlert size={14} />, className: 'text-danger' },
  [VerificationStatus.RUNNING]: {
    icon: <RefreshCw className="animate-spin" size={14} />,
    className: 'font-bold text-accent',
  },
  [VerificationStatus.PENDING]: { icon: <Clock3 size={14} />, className: 'text-muted' },
  [VerificationStatus.CANCELED]: { icon: <CircleX size={14} />, className: 'text-muted' },
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
