import { InfoCircleOutlined } from '@ant-design/icons';
import { Input, Radio, Tooltip } from 'antd';
import { useTranslation } from 'react-i18next';

import type { Notifier } from '../../../../../entity/notifiers';

interface Props {
  notifier: Notifier;
  setNotifier: (notifier: Notifier) => void;
  setUnsaved: () => void;
}

export function EditSESNotifierComponent({ notifier, setNotifier, setUnsaved }: Props) {
  const { t } = useTranslation();
  const sesNotifier = notifier?.sesNotifier;
  const authType = sesNotifier?.authType || 'IAM_ROLE';

  return (
    <>
      <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
        <div className="mb-1 min-w-[150px] sm:mb-0 sm:pr-2">{t('notifiers.ses.fromEmail')}</div>
        <div className="flex items-center">
          <Input
            value={sesNotifier?.fromEmail || ''}
            onChange={(e) => {
              if (!sesNotifier) return;

              setNotifier({
                ...notifier,
                sesNotifier: {
                  ...sesNotifier,
                  fromEmail: e.target.value.trim(),
                },
              });
              setUnsaved();
            }}
            size="small"
            className="w-full max-w-[250px]"
            // eslint-disable-next-line i18next/no-literal-string -- example email address
            placeholder="sender@example.com"
          />

          <Tooltip className="cursor-pointer" title={t('notifiers.ses.fromEmailTooltip')}>
            <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
          </Tooltip>
        </div>
      </div>

      <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
        <div className="mb-1 min-w-[150px] sm:mb-0 sm:pr-2">{t('notifiers.ses.targetEmail')}</div>
        <Input
          value={sesNotifier?.targetEmail || ''}
          onChange={(e) => {
            if (!sesNotifier) return;

            setNotifier({
              ...notifier,
              sesNotifier: {
                ...sesNotifier,
                targetEmail: e.target.value.trim(),
              },
            });
            setUnsaved();
          }}
          size="small"
          className="w-full max-w-[250px]"
          // eslint-disable-next-line i18next/no-literal-string -- example email address
          placeholder="admin@example.com"
        />
      </div>

      <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
        <div className="mb-1 min-w-[150px] sm:mb-0 sm:pr-2">{t('notifiers.ses.region')}</div>
        <Input
          value={sesNotifier?.region || ''}
          onChange={(e) => {
            if (!sesNotifier) return;

            setNotifier({
              ...notifier,
              sesNotifier: {
                ...sesNotifier,
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
        <div className="mb-1 min-w-[150px] sm:mb-0 sm:pr-2">{t('notifiers.ses.authType')}</div>
        <Radio.Group
          value={authType}
          onChange={(e) => {
            if (!sesNotifier) return;

            setNotifier({
              ...notifier,
              sesNotifier: {
                ...sesNotifier,
                authType: e.target.value,
              },
            });
            setUnsaved();
          }}
          size="small"
        >
          <Radio.Button value="IAM_ROLE">{t('notifiers.ses.authTypes.iamRole')}</Radio.Button>
          <Radio.Button value="KEYS">{t('notifiers.ses.authTypes.keys')}</Radio.Button>
        </Radio.Group>
      </div>

      {authType === 'KEYS' ? (
        <>
          <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
            <div className="mb-1 min-w-[150px] sm:mb-0 sm:pr-2">{t('notifiers.ses.accessKey')}</div>
            <Input.Password
              value={sesNotifier?.accessKey || ''}
              onChange={(e) => {
                if (!sesNotifier) return;

                setNotifier({
                  ...notifier,
                  sesNotifier: {
                    ...sesNotifier,
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
            <div className="mb-1 min-w-[150px] sm:mb-0 sm:pr-2">{t('notifiers.ses.secretKey')}</div>
            <Input.Password
              value={sesNotifier?.secretKey || ''}
              onChange={(e) => {
                if (!sesNotifier) return;

                setNotifier({
                  ...notifier,
                  sesNotifier: {
                    ...sesNotifier,
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
            <div className="mb-1 min-w-[150px] sm:mb-0 sm:pr-2">{t('notifiers.ses.roleArn')}</div>
            <div className="flex items-center">
              <Input
                value={sesNotifier?.roleArn || ''}
                onChange={(e) => {
                  if (!sesNotifier) return;

                  setNotifier({
                    ...notifier,
                    sesNotifier: {
                      ...sesNotifier,
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

              <Tooltip className="cursor-pointer" title={t('notifiers.ses.roleArnTooltip')}>
                <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
              </Tooltip>
            </div>
          </div>

          <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
            <div className="mb-1 min-w-[150px] sm:mb-0 sm:pr-2">
              {t('notifiers.ses.externalId')}
            </div>
            <Input.Password
              value={sesNotifier?.externalId || ''}
              onChange={(e) => {
                if (!sesNotifier) return;

                setNotifier({
                  ...notifier,
                  sesNotifier: {
                    ...sesNotifier,
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

          <div className="mb-2 text-xs text-gray-500">{t('notifiers.ses.iamRoleNotice')}</div>
        </>
      )}
    </>
  );
}
