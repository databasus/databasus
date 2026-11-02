import { DeleteOutlined, InfoCircleOutlined, PlusOutlined } from '@ant-design/icons';
import { Button, Form, Input, Select, Tooltip } from 'antd';
import { useId, useMemo } from 'react';
import { Trans, useTranslation } from 'react-i18next';

import { AddressChangedSecretNoticeComponent, InlineCodeComponent } from '@/shared/ui';

import type { WebhookHeader, WebhookNotifierSettings } from '../../../../../entities/notifiers';
import {
  NOTIFICATION_CATEGORY_LABEL_KEYS,
  NotificationCategory,
  WebhookMethod,
  isWebhookHeaderValueMissing,
  isWebhookHeaderNameValid,
} from '../../../../../entities/notifiers';
import type { TranslationKey } from '../../../../../shared/i18n';
import { toOptionalValue } from '../../../../../shared/lib';
import { DEFAULT_ACCEPTED_CATEGORIES } from '../../../lib/defaultAcceptedCategories';
import {
  EXAMPLE_GET_QUERY,
  getExamplePostBody,
  getExamplePostHeaderLines,
} from '../../../lib/webhookExampleRequest';

interface Props {
  settings: WebhookNotifierSettings;
  isSaved: boolean;
  isAddressChanged: boolean;
  savedHeaders: WebhookHeader[];
  onChange: (settings: WebhookNotifierSettings) => void;
}

// eslint-disable-next-line i18next/no-literal-string -- JSON template with the backend's variables
const DEFAULT_BODY_TEMPLATE = `{
  "heading": "{{heading}}",
  "message": "{{message}}"
}`;

function validateJsonTemplate(template: string): TranslationKey | null {
  if (!template.trim()) {
    return null; // Empty is valid (will use default)
  }

  // Replace placeholders with valid JSON strings before parsing
  const testJson = template.replace(/\{\{heading\}\}/g, 'test').replace(/\{\{message\}\}/g, 'test');

  try {
    JSON.parse(testJson);
    return null;
  } catch (e) {
    if (e instanceof SyntaxError) {
      return 'notifiers.webhook.bodyTemplate.invalidJsonFormat';
    }
    return 'notifiers.webhook.bodyTemplate.invalidJson';
  }
}

export function EditWebhookNotifierComponent({
  settings,
  isSaved,
  isAddressChanged,
  savedHeaders,
  onChange,
}: Props) {
  const { t } = useTranslation();
  const noticeIdPrefix = useId();
  const headerFieldId = useId();
  const headers = settings.headers;
  const bodyTemplate = settings.bodyTemplate ?? '';

  const jsonErrorKey = useMemo(() => validateJsonTemplate(bodyTemplate), [bodyTemplate]);

  const acceptedCategories =
    settings.acceptedCategories.length > 0
      ? settings.acceptedCategories
      : DEFAULT_ACCEPTED_CATEGORIES;

  const updateWebhookSettings = (changes: Partial<WebhookNotifierSettings>) =>
    onChange({ ...settings, ...changes });

  const changeAcceptedCategories = (selected: NotificationCategory[]) => {
    const isAllSelected =
      selected.length === 0 || selected[selected.length - 1] === NotificationCategory.ALL;
    updateWebhookSettings({
      acceptedCategories: isAllSelected
        ? [NotificationCategory.ALL]
        : selected.filter((category) => category !== NotificationCategory.ALL),
    });
  };

  const addHeader = () => {
    updateWebhookSettings({ headers: [...headers, { name: '' }] });
  };

  const updateHeader = (index: number, changes: Partial<WebhookHeader>) => {
    const changedHeaders = [...headers];
    changedHeaders[index] = { ...changedHeaders[index], ...changes };
    updateWebhookSettings({ headers: changedHeaders });
  };

  // Leaving a field with nothing to trim must not change the draft, because any draft change
  // withdraws the successful test that Save depends on.
  const trimHeaderName = (index: number) => {
    const trimmedName = headers[index].name.trim();

    if (trimmedName !== headers[index].name) {
      updateHeader(index, { name: trimmedName });
    }
  };

  const removeHeader = (index: number) => {
    updateWebhookSettings({ headers: headers.filter((_, i) => i !== index) });
  };

  return (
    <>
      <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
        <div className="mb-1 min-w-[150px] sm:mb-0 sm:pr-2">{t('notifiers.fields.webhookUrl')}</div>
        <Input
          aria-label={t('notifiers.fields.webhookUrl')}
          value={settings.webhookUrl}
          onChange={(e) => updateWebhookSettings({ webhookUrl: e.target.value.trim() })}
          size="small"
          className="w-full max-w-[250px]"
          placeholder="https://example.com/webhook"
        />
      </div>

      <div className="mt-1 mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
        <div className="mb-1 min-w-[150px] sm:mb-0 sm:pr-2">{t('notifiers.webhook.method')}</div>
        <div className="flex items-center">
          <Select
            aria-label={t('notifiers.webhook.method')}
            value={settings.webhookMethod}
            onChange={(webhookMethod) => updateWebhookSettings({ webhookMethod })}
            size="small"
            className="w-[100px] max-w-[250px]"
            options={[
              { value: WebhookMethod.POST, label: WebhookMethod.POST },
              { value: WebhookMethod.GET, label: WebhookMethod.GET },
            ]}
          />
        </div>
      </div>

      <div className="mt-1 mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
        <div className="mb-1 min-w-[150px] sm:mb-0 sm:pr-2">{t('notifiers.webhook.sendOn')}</div>
        <Select
          aria-label={t('notifiers.webhook.sendOn')}
          mode="multiple"
          value={acceptedCategories}
          onChange={(categories: NotificationCategory[]) => changeAcceptedCategories(categories)}
          size="small"
          className="w-full max-w-[250px]"
          options={Object.values(NotificationCategory).map((category: NotificationCategory) => ({
            label: t(NOTIFICATION_CATEGORY_LABEL_KEYS[category]),
            value: category,
          }))}
          placeholder={t('notifiers.webhook.sendOnPlaceholder')}
        />
      </div>

      <div className="mt-3 mb-1 flex w-full flex-col items-start">
        <div className="mb-1 flex items-center">
          <span className="min-w-[150px] pr-2">
            {t('notifiers.webhook.customHeaders')}{' '}
            <Tooltip title={t('notifiers.webhook.customHeadersTooltip')}>
              <InfoCircleOutlined className="ml-1" />
            </Tooltip>
          </span>
        </div>

        {isSaved && !isAddressChanged && (
          <div className="mb-1 text-xs text-warning">
            {t('notifiers.webhook.savedHeadersHidden')}
          </div>
        )}

        <div className="w-full max-w-[500px]">
          {headers.map((header: WebhookHeader, index: number) => {
            const isNameValid = isWebhookHeaderNameValid(header.name);
            const isValueRequiredAgain =
              isAddressChanged && isWebhookHeaderValueMissing(header, savedHeaders);
            const valueNoticeId = `${noticeIdPrefix}-${index}`;

            return (
              <div key={index} className="mb-1 flex min-w-0 flex-wrap items-start gap-2">
                <Form.Item
                  className="!mb-0 min-w-0 flex-1"
                  validateStatus={isNameValid ? undefined : 'error'}
                  help={
                    isNameValid ? undefined : (
                      <span className="text-danger" id={`${headerFieldId}-${index}-name-error`}>
                        {t('notifiers.webhook.headerNameInvalid')}
                      </span>
                    )
                  }
                >
                  <Input
                    aria-label={t('notifiers.webhook.headerNamePlaceholder')}
                    aria-invalid={!isNameValid}
                    aria-describedby={
                      isNameValid ? undefined : `${headerFieldId}-${index}-name-error`
                    }
                    value={header.name}
                    onChange={(e) => updateHeader(index, { name: e.target.value })}
                    onBlur={() => trimHeaderName(index)}
                    size="small"
                    className="w-full min-w-0"
                    placeholder={t('notifiers.webhook.headerNamePlaceholder')}
                  />
                </Form.Item>
                <div className="min-w-0 flex-1">
                  <Input
                    aria-label={t('notifiers.webhook.headerValuePlaceholder')}
                    aria-describedby={isValueRequiredAgain ? valueNoticeId : undefined}
                    value={header.value ?? ''}
                    onChange={(e) =>
                      updateHeader(index, {
                        value: toOptionalValue(e.target.value),
                      })
                    }
                    size="small"
                    className="w-full"
                    placeholder={t('notifiers.webhook.headerValuePlaceholder')}
                  />
                  {isValueRequiredAgain && (
                    <AddressChangedSecretNoticeComponent id={valueNoticeId} />
                  )}
                </div>
                <Button
                  type="text"
                  danger
                  size="small"
                  aria-label={t('common.actions.remove')}
                  icon={<DeleteOutlined aria-hidden />}
                  onClick={() => removeHeader(index)}
                />
              </div>
            );
          })}

          <Button
            type="dashed"
            size="small"
            icon={<PlusOutlined />}
            onClick={addHeader}
            className="mt-1 !h-auto min-h-6 max-w-full !whitespace-normal"
          >
            {t('notifiers.webhook.addHeader')}
          </Button>
        </div>
      </div>

      {settings.webhookMethod === WebhookMethod.POST && (
        <div className="mt-3 mb-1 flex w-full flex-col items-start">
          <div className="mb-1 flex items-center">
            <span className="min-w-[150px] pr-2">{t('notifiers.webhook.bodyTemplate.title')}</span>
          </div>

          <div className="mb-2 w-full min-w-0 text-xs break-words text-muted">
            <span className="mr-4">
              <Trans
                i18nKey="notifiers.webhook.bodyTemplate.headingVariable"
                components={{
                  variable: <InlineCodeComponent>{'{{heading}}'}</InlineCodeComponent>,
                }}
              />
            </span>
            <span>
              <Trans
                i18nKey="notifiers.webhook.bodyTemplate.messageVariable"
                components={{
                  variable: <InlineCodeComponent>{'{{message}}'}</InlineCodeComponent>,
                }}
              />
            </span>
          </div>

          <Input.TextArea
            aria-label={t('notifiers.webhook.bodyTemplate.title')}
            value={bodyTemplate}
            onChange={(e) =>
              updateWebhookSettings({
                bodyTemplate: toOptionalValue(e.target.value),
              })
            }
            className="w-full max-w-[500px] font-mono text-xs"
            rows={6}
            placeholder={DEFAULT_BODY_TEMPLATE}
            status={jsonErrorKey ? 'error' : undefined}
          />
          {jsonErrorKey && <div className="mt-1 text-xs text-danger">{t(jsonErrorKey)}</div>}
        </div>
      )}

      {settings.webhookUrl && (
        <div className="mt-4">
          <div className="mb-1 font-medium">{t('notifiers.webhook.exampleRequest.title')}</div>

          {settings.webhookMethod === WebhookMethod.GET && (
            <div className="rounded bg-surface p-2 px-3 text-sm break-all">
              <div className="font-semibold text-accent">GET</div>
              <div className="mt-1 !h-auto min-h-6 max-w-full !whitespace-normal">
                {settings.webhookUrl}
                {EXAMPLE_GET_QUERY}
              </div>
              {headers.length > 0 && (
                <div className="mt-2 border-t border-border pt-2">
                  <div className="text-xs font-semibold text-muted">
                    {t('notifiers.webhook.exampleRequest.headers')}
                  </div>

                  {headers
                    .filter((header) => header.name)
                    .map((header, i) => (
                      <div key={i} className="text-xs">
                        {header.name}: {header.value || t('notifiers.webhook.hiddenHeaderValue')}
                      </div>
                    ))}
                </div>
              )}
            </div>
          )}

          {settings.webhookMethod === WebhookMethod.POST && (
            <div className="rounded bg-surface p-2 px-3 font-mono text-sm break-words whitespace-pre-wrap">
              <div className="font-semibold text-accent">POST {settings.webhookUrl}</div>
              <div className="mt-1 text-muted">{getExamplePostHeaderLines(headers)}</div>
              <div className="mt-2 break-words whitespace-pre-wrap">
                {getExamplePostBody(settings.bodyTemplate)}
              </div>
            </div>
          )}
        </div>
      )}
    </>
  );
}
