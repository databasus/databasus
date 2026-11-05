import { useTranslation } from 'react-i18next';

interface Props {
  id: string;
}

export const AddressChangedSecretNoticeComponent = ({ id }: Props) => {
  const { t } = useTranslation();

  return (
    <div id={id} className="mt-1 text-xs text-danger">
      {t('common.messages.addressChangedEnterSecretAgain')}
    </div>
  );
};
