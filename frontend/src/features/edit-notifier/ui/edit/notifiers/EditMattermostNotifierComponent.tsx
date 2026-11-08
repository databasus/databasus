import { Checkbox, TextInput, Tooltip } from '@mantine/core';
import { Info } from 'lucide-react';
import { useId, useState } from 'react';
import { useTranslation } from 'react-i18next';

import {
  MATTERMOST_DELIVERY_MODE_LABEL_KEYS,
  MattermostDeliveryMode,
  type MattermostNotifierSettings,
} from '@/entities/notifiers';
import { getWebsitePageUrl, useLocale } from '@/shared/i18n';
import { toOptionalValue } from '@/shared/lib';
import { WrappingSelect } from '@/shared/ui';
import { AddressChangedSecretNoticeComponent, AdvancedSettingsToggleComponent } from '@/shared/ui';

interface Props {
  settings: MattermostNotifierSettings;
  isSaved: boolean;
  isAddressChanged: boolean;
  onChange: (settings: MattermostNotifierSettings) => void;
}

export function EditMattermostNotifierComponent({
  settings,
  isSaved,
  isAddressChanged,
  onChange,
}: Props) {
  const { t } = useTranslation();
  const { locale } = useLocale();
  const optionalRegionId = useId();
  const botTokenNoticeId = useId();

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
  const isBotTokenRequiredAgain = isAddressChanged && !settings.botToken;

  return (
    <>
      <div className="mb-1 max-w-[250px] sm:ml-[150px]" style={{ lineHeight: 1 }}>
        <a
          className="text-xs !text-accent"
          href={getWebsitePageUrl('notifiersMattermost', locale)}
          target="_blank"
          rel="noreferrer"
        >
          {t('notifiers.mattermost.howToConnect')}
        </a>
      </div>

      <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
        <div className="mb-1 min-w-0 wrap-anywhere sm:mb-0 sm:w-[150px] sm:shrink-0 sm:pr-2">
          {t('notifiers.mattermost.connectVia')}
        </div>
        <WrappingSelect
          aria-label={t('notifiers.mattermost.connectVia')}
          value={settings.deliveryMode}
          data={Object.values(MattermostDeliveryMode).map(
            (deliveryMode: MattermostDeliveryMode) => ({
              label: t(MATTERMOST_DELIVERY_MODE_LABEL_KEYS[deliveryMode]),
              value: deliveryMode,
            }),
          )}
          onChange={(deliveryMode) => updateMattermostSettings({ deliveryMode })}
          size="xs"
          className="w-full max-w-[250px]"
        />
      </div>

      {isWebhookMode ? (
        <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
          <div className="mb-1 min-w-0 wrap-anywhere sm:mb-0 sm:w-[150px] sm:shrink-0 sm:pr-2">
            {t('notifiers.mattermost.incomingWebhookUrl')}
          </div>
          <TextInput
            aria-label={t('notifiers.mattermost.incomingWebhookUrl')}
            value={settings.webhookUrl ?? ''}
            onChange={(e) =>
              updateMattermostSettings({
                webhookUrl: toOptionalValue(e.target.value.trim()),
              })
            }
            size="xs"
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
            <div className="mb-1 min-w-0 wrap-anywhere sm:mb-0 sm:w-[150px] sm:shrink-0 sm:pr-2">
              {t('notifiers.mattermost.serverUrl')}
            </div>
            <TextInput
              aria-label={t('notifiers.mattermost.serverUrl')}
              value={settings.serverUrl ?? ''}
              onChange={(e) =>
                updateMattermostSettings({
                  serverUrl: toOptionalValue(e.target.value.trim()),
                })
              }
              size="xs"
              className="w-full max-w-[250px]"
              placeholder="https://mattermost.example.com"
            />
          </div>

          <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
            <div className="mb-1 min-w-0 wrap-anywhere sm:mb-0 sm:w-[150px] sm:shrink-0 sm:pr-2">
              {t('notifiers.fields.botToken')}
            </div>
            <div className="w-full max-w-[250px]">
              <TextInput
                aria-label={t('notifiers.fields.botToken')}
                value={settings.botToken ?? ''}
                onChange={(e) =>
                  updateMattermostSettings({
                    botToken: toOptionalValue(e.target.value.trim()),
                  })
                }
                size="xs"
                className="w-full"
                placeholder={
                  isSaved && !isAddressChanged
                    ? t('notifiers.secretKeptPlaceholder')
                    : // eslint-disable-next-line i18next/no-literal-string -- example token, not copy
                      'xxxxxxxxxxxxxxxxxxxxxxxxxx'
                }
                description={
                  isBotTokenRequiredAgain && (
                    <AddressChangedSecretNoticeComponent id={botTokenNoticeId} />
                  )
                }
              />
            </div>
          </div>

          <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
            <div className="mb-1 min-w-0 wrap-anywhere sm:mb-0 sm:w-[150px] sm:shrink-0 sm:pr-2">
              {t('notifiers.fields.channelId')}
            </div>
            <div className="flex w-full min-w-0 items-center">
              <TextInput
                aria-label={t('notifiers.fields.channelId')}
                value={settings.targetChannelId ?? ''}
                onChange={(e) =>
                  updateMattermostSettings({
                    targetChannelId: toOptionalValue(e.target.value.trim()),
                  })
                }
                size="xs"
                className="w-full max-w-[250px]"
                // eslint-disable-next-line i18next/no-literal-string -- example channel ID
                placeholder="8f4ycxjmztbwmcy1o3xasjrxha"
              />

              <Tooltip
                className="cursor-pointer"
                label={t('notifiers.mattermost.channelIdTooltip')}
              >
                <Info className="ml-2 inline-block shrink-0 align-middle" size="1em" />
              </Tooltip>
            </div>
          </div>
        </>
      )}

      <div className="mb-1 max-w-[250px] sm:ml-[150px]">
        <AdvancedSettingsToggleComponent
          label={
            isShowOptional
              ? t('notifiers.mattermost.hideOptionalSettings')
              : t('notifiers.mattermost.showOptionalSettings')
          }
          isShowAdvanced={isShowOptional}
          onToggle={() => setIsShowOptional(!isShowOptional)}
          regionId={optionalRegionId}
        />
      </div>

      {isShowOptional && (
        <div id={optionalRegionId}>
          {isWebhookMode && (
            <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
              <div className="mb-1 min-w-0 wrap-anywhere sm:mb-0 sm:w-[150px] sm:shrink-0 sm:pr-2">
                {t('notifiers.mattermost.channelOverride')}
              </div>
              <div className="flex w-full min-w-0 items-center">
                <TextInput
                  aria-label={t('notifiers.mattermost.channelOverride')}
                  value={settings.targetChannelName ?? ''}
                  onChange={(e) =>
                    updateMattermostSettings({
                      targetChannelName: toOptionalValue(e.target.value.trim()),
                    })
                  }
                  size="xs"
                  className="w-full max-w-[250px]"
                  // eslint-disable-next-line i18next/no-literal-string -- example channel name
                  placeholder="town-square"
                />

                <Tooltip
                  className="cursor-pointer"
                  label={t('notifiers.mattermost.channelOverrideTooltip')}
                >
                  <Info className="ml-2 inline-block shrink-0 align-middle" size="1em" />
                </Tooltip>
              </div>
            </div>
          )}

          <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
            <div className="mb-1 min-w-0 wrap-anywhere sm:mb-0 sm:w-[150px] sm:shrink-0 sm:pr-2">
              {t('notifiers.mattermost.postAsUsername')}
            </div>
            <div className="flex w-full min-w-0 items-center">
              <TextInput
                aria-label={t('notifiers.mattermost.postAsUsername')}
                value={settings.overrideUsername ?? ''}
                onChange={(e) =>
                  updateMattermostSettings({
                    overrideUsername: toOptionalValue(e.target.value.trim()),
                  })
                }
                size="xs"
                className="w-full max-w-[250px]"
                // eslint-disable-next-line i18next/no-literal-string -- product name as an example username
                placeholder="Databasus"
              />

              <Tooltip
                className="cursor-pointer"
                label={t('notifiers.mattermost.postAsUsernameTooltip')}
              >
                <Info className="ml-2 inline-block shrink-0 align-middle" size="1em" />
              </Tooltip>
            </div>
          </div>

          <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
            <div className="mb-1 min-w-0 wrap-anywhere sm:mb-0 sm:w-[150px] sm:shrink-0 sm:pr-2">
              {t('notifiers.mattermost.postAsIconUrl')}
            </div>
            <div className="flex w-full min-w-0 items-center">
              <TextInput
                aria-label={t('notifiers.mattermost.postAsIconUrl')}
                value={settings.overrideIconUrl ?? ''}
                onChange={(e) =>
                  updateMattermostSettings({
                    overrideIconUrl: toOptionalValue(e.target.value.trim()),
                  })
                }
                size="xs"
                className="w-full max-w-[250px]"
                placeholder="https://databasus.com/icon.png"
              />

              <Tooltip
                className="cursor-pointer"
                label={t('notifiers.mattermost.postAsIconUrlTooltip')}
              >
                <Info className="ml-2 inline-block shrink-0 align-middle" size="1em" />
              </Tooltip>
            </div>
          </div>

          <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
            <div className="mb-1 min-w-0 wrap-anywhere sm:mb-0 sm:w-[150px] sm:shrink-0 sm:pr-2">
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
                label={<>{t('notifiers.fields.skipTls')}</>}
              />

              <Tooltip className="cursor-pointer" label={t('notifiers.mattermost.skipTlsTooltip')}>
                <Info className="ml-2 inline-block shrink-0 align-middle" size="1em" />
              </Tooltip>
            </div>
          </div>
        </div>
      )}

      <div className="mt-1 max-w-[250px] text-xs text-muted sm:ml-[150px]">
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
