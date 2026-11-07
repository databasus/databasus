import { useTranslation } from 'react-i18next';

interface Props {
  id: string;
}

export const AddressChangedSecretNoticeComponent = ({ id }: Props) => {
  const { t } = useTranslation();

  return (
    <span id={id} className="mt-1 block text-xs text-danger">
      {t('common.messages.addressChangedEnterSecretAgain')}
    </span>
  );
};
