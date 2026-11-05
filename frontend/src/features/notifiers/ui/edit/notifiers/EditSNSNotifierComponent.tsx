import { InfoCircleOutlined } from '@ant-design/icons';
import { Input, Radio, Tooltip } from 'antd';
import { useTranslation } from 'react-i18next';

import type { Notifier } from '../../../../../entity/notifiers';

interface Props {
  notifier: Notifier;
  setNotifier: (notifier: Notifier) => void;
  setUnsaved: () => void;
}

const extractRegionFromTopicARN = (arn: string): string => {
  const parts = arn.split(':');
  if (
    parts.length >= 6 &&
    parts[0] === 'arn' &&
    parts[1]?.startsWith('aws') &&
    parts[2] === 'sns'
  ) {
    return parts[3];
  }

  return '';
};

export function EditSNSNotifierComponent({ notifier, setNotifier, setUnsaved }: Props) {
  const { t } = useTranslation();
  const snsNotifier = notifier?.snsNotifier;
  const authType = snsNotifier?.authType || 'KEYS';

  return (
    <>
      <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
        <div className="mb-1 min-w-[150px] sm:mb-0 sm:pr-2">{t('notifiers.sns.topicArn')}</div>
        <div className="flex items-center">
          <Input
            value={snsNotifier?.topicArn || ''}
            onChange={(e) => {
              if (!snsNotifier) return;

              const nextArn = e.target.value.trim();
              const autoRegion = extractRegionFromTopicARN(nextArn);

              setNotifier({
                ...notifier,
                snsNotifier: {
                  ...snsNotifier,
                  topicArn: nextArn,
                  region: autoRegion || snsNotifier.region,
                },
              });
              setUnsaved();
            }}
            size="small"
            className="w-full max-w-[250px]"
            // eslint-disable-next-line i18next/no-literal-string -- example SNS topic ARN
            placeholder="arn:aws:sns:us-east-1:123456789012:topic"
          />

          <Tooltip className="cursor-pointer" title={t('notifiers.sns.topicArnTooltip')}>
            <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
          </Tooltip>
        </div>
      </div>

      <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
        <div className="mb-1 min-w-[150px] sm:mb-0 sm:pr-2">{t('notifiers.sns.region')}</div>
        <Input
          value={snsNotifier?.region || ''}
          onChange={(e) => {
            if (!snsNotifier) return;

            setNotifier({
              ...notifier,
              snsNotifier: {
                ...snsNotifier,
                region: e.target.value.trim(),
              },
            });
            setUnsaved();
          }}
          size="small"
          className="w-full max-w-[250px]"
          // eslint-disable-next-line i18next/no-literal-string -- example region
          placeholder="us-east-1"
        />
      </div>

      <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
        <div className="mb-1 min-w-[150px] sm:mb-0 sm:pr-2">{t('notifiers.sns.authType')}</div>
        <Radio.Group
          value={authType}
          onChange={(e) => {
            if (!snsNotifier) return;

            setNotifier({
              ...notifier,
              snsNotifier: {
                ...snsNotifier,
                authType: e.target.value,
              },
            });
            setUnsaved();
          }}
          size="small"
        >
          <Radio.Button value="KEYS">{t('notifiers.sns.authTypes.keys')}</Radio.Button>
          <Radio.Button value="IAM_ROLE">{t('notifiers.sns.authTypes.iamRole')}</Radio.Button>
        </Radio.Group>
      </div>

      {authType === 'KEYS' ? (
        <>
          <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
            <div className="mb-1 min-w-[150px] sm:mb-0 sm:pr-2">{t('notifiers.sns.accessKey')}</div>
            <Input.Password
              value={snsNotifier?.accessKey || ''}
              onChange={(e) => {
                if (!snsNotifier) return;

                setNotifier({
                  ...notifier,
                  snsNotifier: {
                    ...snsNotifier,
                    accessKey: e.target.value.trim(),
                  },
                });
                setUnsaved();
              }}
              size="small"
              className="w-full max-w-[250px]"
              // eslint-disable-next-line i18next/no-literal-string -- example access key
              placeholder="AKIAIOSFODNN7EXAMPLE"
              autoComplete="off"
            />
          </div>

          <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
            <div className="mb-1 min-w-[150px] sm:mb-0 sm:pr-2">{t('notifiers.sns.secretKey')}</div>
            <Input.Password
              value={snsNotifier?.secretKey || ''}
              onChange={(e) => {
                if (!snsNotifier) return;

                setNotifier({
                  ...notifier,
                  snsNotifier: {
                    ...snsNotifier,
                    secretKey: e.target.value.trim(),
                  },
                });
                setUnsaved();
              }}
              size="small"
              className="w-full max-w-[250px]"
              // eslint-disable-next-line i18next/no-literal-string -- example secret key
              placeholder="wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
              autoComplete="off"
            />
          </div>
        </>
      ) : (
        <>
          <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
            <div className="mb-1 min-w-[150px] sm:mb-0 sm:pr-2">{t('notifiers.sns.roleArn')}</div>
            <div className="flex items-center">
              <Input
                value={snsNotifier?.roleArn || ''}
                onChange={(e) => {
                  if (!snsNotifier) return;

                  setNotifier({
                    ...notifier,
                    snsNotifier: {
                      ...snsNotifier,
                      roleArn: e.target.value.trim(),
                    },
                  });
                  setUnsaved();
                }}
                size="small"
                className="w-full max-w-[250px]"
                // eslint-disable-next-line i18next/no-literal-string -- example role ARN
                placeholder="arn:aws:iam::123456789012:role/RoleName"
              />

              <Tooltip className="cursor-pointer" title={t('notifiers.sns.roleArnTooltip')}>
                <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
              </Tooltip>
            </div>
          </div>

          <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
            <div className="mb-1 min-w-[150px] sm:mb-0 sm:pr-2">
              {t('notifiers.sns.externalId')}
            </div>
            <Input.Password
              value={snsNotifier?.externalId || ''}
              onChange={(e) => {
                if (!snsNotifier) return;

                setNotifier({
                  ...notifier,
                  snsNotifier: {
                    ...snsNotifier,
                    externalId: e.target.value.trim(),
                  },
                });
                setUnsaved();
              }}
              size="small"
              className="w-full max-w-[250px]"
              placeholder={t('storages.edit.optionalPlaceholder', {
                // eslint-disable-next-line i18next/no-literal-string -- example external ID
                example: 'my-external-id',
              })}
              autoComplete="off"
            />
          </div>

          <div className="mb-2 text-xs text-gray-500">{t('notifiers.sns.iamRoleNotice')}</div>
        </>
      )}
    </>
  );
}
