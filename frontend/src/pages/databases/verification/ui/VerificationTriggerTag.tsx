import { Badge } from '@mantine/core';
import type { JSX } from 'react';
import { useTranslation } from 'react-i18next';

import {
  type VerificationTrigger,
  VERIFICATION_TRIGGER_LABEL_KEYS,
} from '@/pages/databases/verification/model/VerificationTrigger';

interface Props {
  trigger: VerificationTrigger;
}

export const VerificationTriggerTag = ({ trigger }: Props): JSX.Element => {
  const { t } = useTranslation();

  return (
    <Badge
      variant="default"
      radius="sm"
      size="sm"
      styles={{ label: { textTransform: 'none', fontWeight: 400 } }}
    >
      {t(VERIFICATION_TRIGGER_LABEL_KEYS[trigger])}
    </Badge>
  );
};
