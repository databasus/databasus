import { Tag } from 'antd';
import type { JSX } from 'react';
import { useTranslation } from 'react-i18next';

import type { VerificationTrigger } from '@/pages/databases/model/verification/VerificationTrigger';
import { VERIFICATION_TRIGGER_LABEL_KEYS } from '@/pages/databases/model/verification/VerificationTriggerLabelKeys';

interface Props {
  trigger: VerificationTrigger;
}

export const VerificationTriggerTag = ({ trigger }: Props): JSX.Element => {
  const { t } = useTranslation();

  return <Tag>{t(VERIFICATION_TRIGGER_LABEL_KEYS[trigger])}</Tag>;
};
