import { useTranslation } from 'react-i18next';

import type { Notifier } from '../../../../../entity/notifiers';

interface Props {
  notifier: Notifier;
}

export function ShowSNSNotifierComponent({ notifier }: Props) {
  const { t } = useTranslation();
  const snsNotifier = notifier?.snsNotifier;
  const isIAMRole = snsNotifier?.authType === 'IAM_ROLE';

  return (
    <>
      <div className="mb-1 flex items-center">
        <div className="min-w-[110px] pr-2">{t('notifiers.sns.topicArn')}</div>
        {snsNotifier?.topicArn}
      </div>

      <div className="mb-1 flex items-center">
        <div className="min-w-[110px] pr-2">{t('notifiers.sns.region')}</div>
        {snsNotifier?.region}
      </div>

      <div className="mb-1 flex items-center">
        <div className="min-w-[110px] pr-2">{t('notifiers.sns.authType')}</div>
        {isIAMRole ? t('notifiers.sns.authTypes.iamRole') : t('notifiers.sns.authTypes.keys')}
      </div>

      {isIAMRole ? (
        snsNotifier?.roleArn ? (
          <div className="mb-1 flex items-center">
            <div className="min-w-[110px] pr-2">{t('notifiers.sns.roleArn')}</div>
            {snsNotifier.roleArn}
          </div>
        ) : null
      ) : (
        <>
          <div className="mb-1 flex items-center">
            <div className="min-w-[110px] pr-2">{t('notifiers.sns.accessKey')}</div>
            {'*************'}
          </div>

          <div className="mb-1 flex items-center">
            <div className="min-w-[110px] pr-2">{t('notifiers.sns.secretKey')}</div>
            {'*************'}
          </div>
        </>
      )}
    </>
  );
}
