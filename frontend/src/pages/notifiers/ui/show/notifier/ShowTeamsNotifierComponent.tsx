import { useTranslation } from 'react-i18next';

export function ShowTeamsNotifierComponent() {
  const { t } = useTranslation();

  return (
    <div className="flex items-center">
      <div className="min-w-27.5 pr-2">{t('notifiers.teams.powerAutomateUrlLabel')}</div>
      <div>*********</div>
    </div>
  );
}
