import { useTranslation } from 'react-i18next';

import type { Notifier } from '../../../../../entity/notifiers';

interface Props {
  notifier: Notifier;
}

export function ShowSESNotifierComponent({ notifier }: Props) {
  const { t } = useTranslation();
  const sesNotifier = notifier?.sesNotifier;
  const isIAMRole = sesNotifier?.authType === 'IAM_ROLE';

  return (
    <>
      <div className="mb-1 flex items-center">
        <div className="min-w-[110px] pr-2">{t('notifiers.ses.fromEmail')}</div>
        {sesNotifier?.fromEmail}
      </div>

      <div className="mb-1 flex items-center">
        <div className="min-w-[110px] pr-2">{t('notifiers.ses.targetEmail')}</div>
        {sesNotifier?.targetEmail}
      </div>

      <div className="mb-1 flex items-center">
        <div className="min-w-[110px] pr-2">{t('notifiers.ses.region')}</div>
        {sesNotifier?.region}
      </div>

      <div className="mb-1 flex items-center">
        <div className="min-w-[110px] pr-2">{t('notifiers.ses.authType')}</div>
        {isIAMRole ? t('notifiers.ses.authTypes.iamRole') : t('notifiers.ses.authTypes.keys')}
      </div>

      {isIAMRole ? (
        sesNotifier?.roleArn ? (
          <div className="mb-1 flex items-center">
            <div className="min-w-[110px] pr-2">{t('notifiers.ses.roleArn')}</div>
            {sesNotifier.roleArn}
          </div>
        ) : null
      ) : (
        <>
          <div className="mb-1 flex items-center">
            <div className="min-w-[110px] pr-2">{t('notifiers.ses.accessKey')}</div>
            {'*************'}
          </div>

          <div className="mb-1 flex items-center">
            <div className="min-w-[110px] pr-2">{t('notifiers.ses.secretKey')}</div>
            {'*************'}
          </div>
        </>
      )}
    </>
  );
}
