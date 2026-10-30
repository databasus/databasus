import { InfoCircleOutlined } from '@ant-design/icons';
import { Checkbox, Input, Select, Tooltip } from 'antd';
import { useState } from 'react';
import { useTranslation } from 'react-i18next';

import {
  MATTERMOST_DELIVERY_MODE_LABEL_KEYS,
  MattermostDeliveryMode,
  type MattermostNotifierSettings,
} from '../../../../../entities/notifiers';
import { getWebsitePageUrl, useLocale } from '../../../../../shared/i18n';
import { toOptionalValue } from '../../../../../shared/lib';

interface Props {
  settings: MattermostNotifierSettings;
  isSaved: boolean;
  onChange: (settings: MattermostNotifierSettings) => void;
}

export function EditMattermostNotifierComponent({ settings, isSaved, onChange }: Props) {
  const { t } = useTranslation();
  const { locale } = useLocale();

  const [isShowOptional, setIsShowOptional] = useState(
    () =>
      !!(
        settings.targetChannelName ||
        settings.overrideUsername ||
        settings.overrideIconUrl ||
        settings.isInsecureSkipVerify
      ),
  );

  const updateMattermostSettings = (changes: Partial<MattermostNotifierSettings>) =>
    onChange({ ...settings, ...changes });

  const isWebhookMode = settings.deliveryMode === MattermostDeliveryMode.WEBHOOK;

  return (
    <>
      <div className="mb-1 max-w-[250px] sm:ml-[150px]" style={{ lineHeight: 1 }}>
        <a
          className="text-xs !text-blue-600"
          href={getWebsitePageUrl('notifiersMattermost', locale)}
          target="_blank"
          rel="noreferrer"
        >
          {t('notifiers.mattermost.howToConnect')}
        </a>
      </div>

      <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
        <div className="mb-1 min-w-[150px] sm:mb-0 sm:pr-2">
          {t('notifiers.mattermost.connectVia')}
        </div>
        <Select
          value={settings.deliveryMode}
          options={Object.values(MattermostDeliveryMode).map(
            (deliveryMode: MattermostDeliveryMode) => ({
              label: t(MATTERMOST_DELIVERY_MODE_LABEL_KEYS[deliveryMode]),
              value: deliveryMode,
            }),
          )}
          onChange={(deliveryMode) => updateMattermostSettings({ deliveryMode })}
          size="small"
          className="w-full max-w-[250px]"
        />
      </div>

      {isWebhookMode ? (
        <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
          <div className="mb-1 min-w-[150px] sm:mb-0 sm:pr-2">
            {t('notifiers.mattermost.incomingWebhookUrl')}
          </div>
          <Input
            value={settings.webhookUrl ?? ''}
            onChange={(e) =>
              updateMattermostSettings({
                webhookUrl: toOptionalValue(e.target.value.trim()),
              })
            }
            size="small"
            className="w-full max-w-[250px]"
            placeholder={
              isSaved
                ? t('notifiers.secretKeptPlaceholder')
                : 'https://mattermost.example.com/hooks/xxxxxxxx'
            }
          />
        </div>
      ) : (
        <>
          <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
            <div className="mb-1 min-w-[150px] sm:mb-0 sm:pr-2">
              {t('notifiers.mattermost.serverUrl')}
            </div>
            <Input
              value={settings.serverUrl ?? ''}
              onChange={(e) =>
                updateMattermostSettings({
                  serverUrl: toOptionalValue(e.target.value.trim()),
                })
              }
              size="small"
              className="w-full max-w-[250px]"
              placeholder="https://mattermost.example.com"
            />
          </div>

          <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
            <div className="mb-1 min-w-[150px] sm:mb-0 sm:pr-2">
              {t('notifiers.fields.botToken')}
            </div>
            <Input
              value={settings.botToken ?? ''}
              onChange={(e) =>
                updateMattermostSettings({
                  botToken: toOptionalValue(e.target.value.trim()),
                })
              }
              size="small"
              className="w-full max-w-[250px]"
              placeholder={
                isSaved
                  ? t('notifiers.secretKeptPlaceholder')
                  : // eslint-disable-next-line i18next/no-literal-string -- example token, not copy
                    'xxxxxxxxxxxxxxxxxxxxxxxxxx'
              }
            />
          </div>

          <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
            <div className="mb-1 min-w-[150px] sm:mb-0 sm:pr-2">
              {t('notifiers.fields.channelId')}
            </div>
            <div className="flex items-center">
              <Input
                value={settings.targetChannelId ?? ''}
                onChange={(e) =>
                  updateMattermostSettings({
                    targetChannelId: toOptionalValue(e.target.value.trim()),
                  })
                }
                size="small"
                className="w-full max-w-[250px]"
                // eslint-disable-next-line i18next/no-literal-string -- example channel ID
                placeholder="8f4ycxjmztbwmcy1o3xasjrxha"
              />

              <Tooltip
                className="cursor-pointer"
                title={t('notifiers.mattermost.channelIdTooltip')}
              >
                <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
              </Tooltip>
            </div>
          </div>
        </>
      )}

      <div className="mb-1 max-w-[250px] sm:ml-[150px]">
        <button
          type="button"
          onClick={() => setIsShowOptional(!isShowOptional)}
          className="text-xs text-blue-600 hover:underline"
        >
          {isShowOptional
            ? t('notifiers.mattermost.hideOptionalSettings')
            : t('notifiers.mattermost.showOptionalSettings')}
        </button>
      </div>

      {isShowOptional && (
        <>
          {isWebhookMode && (
            <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
              <div className="mb-1 min-w-[150px] sm:mb-0 sm:pr-2">
                {t('notifiers.mattermost.channelOverride')}
              </div>
              <div className="flex items-center">
                <Input
                  value={settings.targetChannelName ?? ''}
                  onChange={(e) =>
                    updateMattermostSettings({
                      targetChannelName: toOptionalValue(e.target.value.trim()),
                    })
                  }
                  size="small"
                  className="w-full max-w-[250px]"
                  // eslint-disable-next-line i18next/no-literal-string -- example channel name
                  placeholder="town-square"
                />

                <Tooltip
                  className="cursor-pointer"
                  title={t('notifiers.mattermost.channelOverrideTooltip')}
                >
                  <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
                </Tooltip>
              </div>
            </div>
          )}

          <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
            <div className="mb-1 min-w-[150px] sm:mb-0 sm:pr-2">
              {t('notifiers.mattermost.postAsUsername')}
            </div>
            <div className="flex items-center">
              <Input
                value={settings.overrideUsername ?? ''}
                onChange={(e) =>
                  updateMattermostSettings({
                    overrideUsername: toOptionalValue(e.target.value.trim()),
                  })
                }
                size="small"
                className="w-full max-w-[250px]"
                // eslint-disable-next-line i18next/no-literal-string -- product name as an example username
                placeholder="Databasus"
              />

              <Tooltip
                className="cursor-pointer"
                title={t('notifiers.mattermost.postAsUsernameTooltip')}
              >
                <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
              </Tooltip>
            </div>
          </div>

          <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
            <div className="mb-1 min-w-[150px] sm:mb-0 sm:pr-2">
              {t('notifiers.mattermost.postAsIconUrl')}
            </div>
            <div className="flex items-center">
              <Input
                value={settings.overrideIconUrl ?? ''}
                onChange={(e) =>
                  updateMattermostSettings({
                    overrideIconUrl: toOptionalValue(e.target.value.trim()),
                  })
                }
                size="small"
                className="w-full max-w-[250px]"
                placeholder="https://databasus.com/icon.png"
              />

              <Tooltip
                className="cursor-pointer"
                title={t('notifiers.mattermost.postAsIconUrlTooltip')}
              >
                <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
              </Tooltip>
            </div>
          </div>

          <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
            <div className="mb-1 min-w-[150px] sm:mb-0 sm:pr-2">
              {t('notifiers.fields.skipTlsVerify')}
            </div>
            <div className="flex items-center">
              <Checkbox
                checked={settings.isInsecureSkipVerify}
                onChange={(e) =>
                  updateMattermostSettings({
                    isInsecureSkipVerify: e.target.checked,
                  })
                }
              >
                {t('notifiers.fields.skipTls')}
              </Checkbox>

              <Tooltip className="cursor-pointer" title={t('notifiers.mattermost.skipTlsTooltip')}>
                <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
              </Tooltip>
            </div>
          </div>
        </>
      )}

      <div className="mt-1 max-w-[250px] text-xs text-gray-500 sm:ml-[150px] dark:text-gray-400">
        {isWebhookMode ? (
          <>
            <strong>{t('notifiers.mattermost.webhookHowTo.title')}</strong>
            <br />
            <br />
            {t('notifiers.mattermost.webhookHowTo.openIncomingWebhooks')}
            <br />
            {t('notifiers.mattermost.webhookHowTo.addWebhook')}
            <br />
            {t('notifiers.mattermost.webhookHowTo.copyUrl')}
          </>
        ) : (
          <>
            <strong>{t('notifiers.mattermost.botHowTo.title')}</strong>
            <br />
            <br />
            {t('notifiers.mattermost.botHowTo.addBotAccount')}
            <br />
            {t('notifiers.mattermost.botHowTo.copyToken')}
            <br />
            {t('notifiers.mattermost.botHowTo.addBotToChannel')}
          </>
        )}
      </div>
    </>
  );
}
