import { Button } from 'antd';
import { useId } from 'react';
import { useTranslation } from 'react-i18next';

interface Props {
  subjectId: string;
  className: string;
  onClick: () => void;
}

export const EditIconButton = ({ subjectId, className, onClick }: Props) => {
  const { t } = useTranslation();
  const buttonId = useId();

  return (
    <Button
      id={buttonId}
      type="text"
      className={`!min-w-0 !p-0 ${className}`}
      aria-label={t('common.actions.edit')}
      aria-labelledby={`${buttonId} ${subjectId}`}
      onClick={onClick}
    >
      <img src="/icons/pen-gray.svg" alt="" />
    </Button>
  );
};
