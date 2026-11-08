import { Button, Switch, TextInput, Tooltip } from '@mantine/core';
import { Info } from 'lucide-react';
import { useId, useState } from 'react';
import { Trans, useTranslation } from 'react-i18next';

import type { TelegramNotifierSettings } from '@/entities/notifiers';
import { toOptionalValue } from '@/shared/lib';
import {
  AddressChangedSecretNoticeComponent,
  AdvancedFieldRowComponent,
  AdvancedSettingsToggleComponent,
} from '@/shared/ui';

interface Props {
  settings: TelegramNotifierSettings;
  isSaved: boolean;
  isAddressChanged: boolean;
  onChange: (settings: TelegramNotifierSettings) => void;
}

// A thread ID of 0 keeps the topic switched on while its ID is not entered, and the form refuses
// to save it.
const TOPIC_WITHOUT_ID = 0;

const chatIdBotLink = <a href="https://t.me/getmyid_bot" target="_blank" rel="noreferrer" />;

export function EditTelegramNotifierComponent({
  settings,
  isSaved,
  isAddressChanged,
  onChange,
}: Props) {
  const { t } = useTranslation();
  const fieldPrefix = useId();
  const [isShowHowToGetChatId, setIsShowHowToGetChatId] = useState(false);
  const [isShowAdvanced, setIsShowAdvanced] = useState(
    !!settings.botApiUrl || settings.isProxyEnabled,
  );

  const isBotTokenRequiredAgain = isAddressChanged && !settings.botToken;
  const isSendToThreadEnabled = settings.threadId !== undefined;

  return (
    <>
      <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
        <div className="mb-1 min-w-0 wrap-anywhere sm:mb-0 sm:w-[150px] sm:shrink-0 sm:pr-2">
          {t('notifiers.fields.botToken')}
        </div>
        <div className="w-full max-w-[250px]">
          <TextInput
            aria-label={t('notifiers.fields.botToken')}
            value={settings.botToken ?? ''}
            onChange={(e) =>
              onChange({
                ...settings,
                botToken: toOptionalValue(e.target.value.trim()),
              })
            }
            size="xs"
            className="w-full"
            placeholder={
              isSaved && !isAddressChanged
                ? t('notifiers.secretKeptPlaceholder')
                : // eslint-disable-next-line i18next/no-literal-string -- example token, not copy
                  '1234567890:ABCDEFGHIJKLMNOPQRSTUVWXYZ'
            }
            description={
              isBotTokenRequiredAgain && (
                <AddressChangedSecretNoticeComponent id={`${fieldPrefix}-botToken-notice`} />
              )
            }
          />
        </div>
      </div>

      <div className="mb-1 sm:ml-[150px]">
        <a
          className="text-xs !text-accent"
          href="https://www.siteguarding.com/en/how-to-get-telegram-bot-api-token"
          target="_blank"
          rel="noreferrer"
        >
          {t('notifiers.telegram.botTokenHelpLink')}
        </a>
      </div>

      <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
        <div className="mb-1 min-w-0 wrap-anywhere sm:mb-0 sm:w-[150px] sm:shrink-0 sm:pr-2">
          {t('notifiers.fields.targetChatId')}
        </div>
        <div className="flex w-full min-w-0 items-center">
          <TextInput
            aria-label={t('notifiers.fields.targetChatId')}
            value={settings.targetChatId}
            onChange={(e) => onChange({ ...settings, targetChatId: e.target.value.trim() })}
            size="xs"
            className="w-full max-w-[250px]"
            placeholder="-1001234567890"
          />

          <Tooltip className="cursor-pointer" label={t('notifiers.telegram.targetChatIdTooltip')}>
            <Info className="ml-2 inline-block shrink-0 align-middle text-muted" size="1em" />
          </Tooltip>
        </div>
      </div>

      <div className="max-w-[250px] sm:ml-[150px]">
        {!isShowHowToGetChatId ? (
          <Button
            variant="transparent"
            className="mt-1 !h-auto !px-0 text-left text-xs !whitespace-normal"
            onClick={() => setIsShowHowToGetChatId(true)}
          >
            {t('notifiers.telegram.chatIdHelp.toggle')}
          </Button>
        ) : (
          <div className="mt-1 text-xs text-muted">
            <Trans
              i18nKey="notifiers.telegram.chatIdHelp.privateChat"
              components={{ botLink: chatIdBotLink, underline: <u /> }}
            />
            <br />
            <br />
            <Trans
              i18nKey="notifiers.telegram.chatIdHelp.group"
              components={{ botLink: chatIdBotLink }}
            />
          </div>
        )}
      </div>

      <div className="mt-4 mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
        <div className="mb-1 min-w-0 wrap-anywhere sm:mb-0 sm:w-[150px] sm:shrink-0 sm:pr-2">
          {t('notifiers.telegram.sendToGroupTopic')}
        </div>
        <div className="flex items-center">
          <Switch
            aria-label={t('notifiers.telegram.sendToGroupTopic')}
            checked={isSendToThreadEnabled}
            onChange={(event) =>
              onChange({
                ...settings,
                threadId: event.currentTarget.checked ? TOPIC_WITHOUT_ID : undefined,
              })
            }
            size="xs"
          />

          <Tooltip
            className="cursor-pointer"
            label={t('notifiers.telegram.sendToGroupTopicTooltip')}
          >
            <Info className="ml-2 inline-block shrink-0 align-middle text-muted" size="1em" />
          </Tooltip>
        </div>
      </div>

      {isSendToThreadEnabled && (
        <>
          <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
            <div className="mb-1 min-w-0 wrap-anywhere sm:mb-0 sm:w-[150px] sm:shrink-0 sm:pr-2">
              {t('notifiers.telegram.threadId')}
            </div>
            <div className="flex w-full min-w-0 items-center">
              <TextInput
                aria-label={t('notifiers.telegram.threadId')}
                value={settings.threadId ? settings.threadId.toString() : ''}
                onChange={(e) => {
                  const threadId = parseInt(e.target.value.trim(), 10);

                  onChange({
                    ...settings,
                    threadId: Number.isNaN(threadId) ? TOPIC_WITHOUT_ID : threadId,
                  });
                }}
                size="xs"
                className="w-full max-w-[250px]"
                placeholder="3"
                type="number"
                min="1"
              />

              <Tooltip className="cursor-pointer" label={t('notifiers.telegram.threadIdTooltip')}>
                <Info className="ml-2 inline-block shrink-0 align-middle text-muted" size="1em" />
              </Tooltip>
            </div>
          </div>

          <div className="max-w-[250px] sm:ml-[150px]">
            <div className="mt-1 text-xs text-muted">
              {t('notifiers.telegram.threadIdHelp.steps')}
              <br />
              <br />
              <Trans
                i18nKey="notifiers.telegram.threadIdHelp.example"
                components={{
                  bold: <strong />,
                  code: <code className="rounded bg-surface px-1" />,
                }}
              />
              <br />
              <br />
              <Trans
                i18nKey="notifiers.telegram.threadIdHelp.note"
                components={{ bold: <strong /> }}
              />
            </div>
          </div>
        </>
      )}

      <AdvancedSettingsToggleComponent
        regionId={`${fieldPrefix}-advanced`}
        label={t('notifiers.advancedSettings')}
        isShowAdvanced={isShowAdvanced}
        onToggle={() => setIsShowAdvanced(!isShowAdvanced)}
      />

      {isShowAdvanced && (
        <div id={`${fieldPrefix}-advanced`}>
          <AdvancedFieldRowComponent
            controlId={`${fieldPrefix}-botApiUrl`}
            label={t('notifiers.telegram.botApiUrl')}
            tooltip={t('notifiers.telegram.botApiUrlTooltip')}
          >
            <TextInput
              id={`${fieldPrefix}-botApiUrl`}
              description={t('notifiers.telegram.botApiUrlTooltip')}
              descriptionProps={{ className: 'sr-only' }}
              classNames={{ wrapper: 'mb-0' }}
              value={settings.botApiUrl ?? ''}
              onChange={(e) =>
                onChange({
                  ...settings,
                  botApiUrl: toOptionalValue(e.target.value.trim()),
                })
              }
              size="xs"
              className="w-full max-w-[250px]"
              // eslint-disable-next-line i18next/no-literal-string -- example URL, not copy
              placeholder="https://api.telegram.org"
            />
          </AdvancedFieldRowComponent>

          <AdvancedFieldRowComponent
            controlId={`${fieldPrefix}-useProxy`}
            label={t('notifiers.telegram.useProxy')}
            tooltip={t('notifiers.telegram.useProxyTooltip')}
          >
            <Switch
              id={`${fieldPrefix}-useProxy`}
              aria-describedby={`${fieldPrefix}-useProxy-help`}
              checked={settings.isProxyEnabled}
              onChange={(event) =>
                onChange({
                  ...settings,
                  isProxyEnabled: event.currentTarget.checked,
                  proxyUrl: event.currentTarget.checked ? settings.proxyUrl : undefined,
                })
              }
              size="xs"
            />
          </AdvancedFieldRowComponent>

          {settings.isProxyEnabled && (
            <AdvancedFieldRowComponent
              controlId={`${fieldPrefix}-proxyUrl`}
              label={t('notifiers.telegram.proxyUrl')}
              tooltip={t('notifiers.telegram.proxyUrlTooltip')}
            >
              <TextInput
                id={`${fieldPrefix}-proxyUrl`}
                description={t('notifiers.telegram.proxyUrlTooltip')}
                descriptionProps={{ className: 'sr-only' }}
                classNames={{ wrapper: 'mb-0' }}
                value={settings.proxyUrl ?? ''}
                onChange={(e) =>
                  onChange({
                    ...settings,
                    proxyUrl: toOptionalValue(e.target.value.trim()),
                  })
                }
                size="xs"
                className="w-full max-w-[250px]"
                placeholder={
                  isSaved
                    ? t('notifiers.secretKeptPlaceholder')
                    : // eslint-disable-next-line i18next/no-literal-string -- example proxy URL
                      'socks5://user:pass@host:1080'
                }
              />
            </AdvancedFieldRowComponent>
          )}
        </div>
      )}
    </>
  );
}
