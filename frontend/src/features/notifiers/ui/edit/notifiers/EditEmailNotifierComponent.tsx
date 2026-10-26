import { InfoCircleOutlined } from '@ant-design/icons';
import { Checkbox, Input, Select, Tooltip } from 'antd';
import { useState } from 'react';
import { useTranslation } from 'react-i18next';

import {
  EMAIL_NOTIFIER_SECURITY_LABEL_KEYS,
  EmailNotifierSecurity,
  type EmailNotifierSettings,
  getDefaultEmailNotifierSecurity,
  getSecurityAfterPortChange,
} from '../../../../../entity/notifiers';
import { toOptionalValue } from '../../../../../shared/lib';
import {
  AdvancedFieldRowComponent,
  AdvancedSettingsToggleComponent,
} from '../../../../../shared/ui';

interface Props {
  settings: EmailNotifierSettings;
  isSaved: boolean;
  onChange: (settings: EmailNotifierSettings) => void;
}

export function EditEmailNotifierComponent({ settings, isSaved, onChange }: Props) {
  const { t } = useTranslation();
  const hasAdvancedValues =
    settings.isInsecureSkipVerify ||
    !!settings.heloName ||
    settings.security !== getDefaultEmailNotifierSecurity(settings.smtpPort);
  const [showAdvanced, setShowAdvanced] = useState(hasAdvancedValues);

  return (
    <>
      <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
        <div className="mb-1 min-w-[150px] sm:mb-0 sm:pr-2">{t('notifiers.email.targetEmail')}</div>
        <div className="flex items-center">
          <Input
            value={settings.targetEmail ?? ''}
            onChange={(e) => onChange({ ...settings, targetEmail: e.target.value.trim() })}
            size="small"
            className="w-full max-w-[250px]"
            placeholder="example@gmail.com"
          />

          <Tooltip className="cursor-pointer" title={t('notifiers.email.targetEmailTooltip')}>
            <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
          </Tooltip>
        </div>
      </div>

      <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
        <div className="mb-1 min-w-[150px] sm:mb-0 sm:pr-2">{t('notifiers.email.smtpHost')}</div>
        <Input
          value={settings.smtpHost ?? ''}
          onChange={(e) => onChange({ ...settings, smtpHost: e.target.value.trim() })}
          size="small"
          className="w-full max-w-[250px]"
          // eslint-disable-next-line i18next/no-literal-string -- example hostname
          placeholder="smtp.gmail.com"
        />
      </div>

      <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
        <div className="mb-1 min-w-[150px] sm:mb-0 sm:pr-2">{t('notifiers.email.smtpPort')}</div>
        <Input
          type="number"
          value={settings.smtpPort ?? ''}
          onChange={(e) => {
            const nextPort = Number(e.target.value);

            onChange({
              ...settings,
              smtpPort: nextPort,
              security: getSecurityAfterPortChange(settings.security, settings.smtpPort, nextPort),
            });
          }}
          size="small"
          className="w-full max-w-[250px]"
          placeholder="25"
        />
      </div>

      <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
        <div className="mb-1 min-w-[150px] sm:mb-0 sm:pr-2">{t('notifiers.email.smtpUser')}</div>
        <Input
          value={settings.smtpUser ?? ''}
          onChange={(e) =>
            onChange({
              ...settings,
              smtpUser: toOptionalValue(e.target.value.trim()),
            })
          }
          size="small"
          className="w-full max-w-[250px]"
          placeholder="user@gmail.com"
        />
      </div>

      <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
        <div className="mb-1 min-w-[150px] sm:mb-0 sm:pr-2">
          {t('notifiers.email.smtpPassword')}
        </div>
        <Input
          type="password"
          value={settings.smtpPassword ?? ''}
          onChange={(e) =>
            onChange({
              ...settings,
              smtpPassword: toOptionalValue(e.target.value.trim()),
            })
          }
          size="small"
          className="w-full max-w-[250px]"
          placeholder={
            isSaved
              ? t('notifiers.secretKeptPlaceholder')
              : t('notifiers.email.smtpPasswordPlaceholder')
          }
        />
      </div>

      <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
        <div className="mb-1 min-w-[150px] sm:mb-0 sm:pr-2">{t('notifiers.email.from')}</div>
        <div className="flex items-center">
          <Input
            value={settings.from ?? ''}
            onChange={(e) =>
              onChange({
                ...settings,
                from: toOptionalValue(e.target.value.trim()),
              })
            }
            size="small"
            className="w-full max-w-[250px]"
            placeholder="example@example.com"
          />

          <Tooltip className="cursor-pointer" title={t('notifiers.email.fromTooltip')}>
            <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
          </Tooltip>
        </div>
      </div>

      <AdvancedSettingsToggleComponent
        label={t('notifiers.advancedSettings')}
        isShowAdvanced={showAdvanced}
        onToggle={() => setShowAdvanced(!showAdvanced)}
      />

      {showAdvanced && (
        <>
          <AdvancedFieldRowComponent
            label={t('notifiers.email.security')}
            tooltip={t('notifiers.email.securityTooltip')}
          >
            <Select
              value={settings.security}
              onChange={(security: EmailNotifierSecurity) => onChange({ ...settings, security })}
              size="small"
              className="w-[250px] max-w-[250px]"
              options={Object.values(EmailNotifierSecurity).map(
                (security: EmailNotifierSecurity) => ({
                  value: security,
                  label: t(EMAIL_NOTIFIER_SECURITY_LABEL_KEYS[security]),
                }),
              )}
            />
          </AdvancedFieldRowComponent>

          <AdvancedFieldRowComponent
            label={t('notifiers.email.heloName')}
            tooltip={t('notifiers.email.heloNameTooltip')}
          >
            <Input
              value={settings.heloName ?? ''}
              onChange={(e) =>
                onChange({
                  ...settings,
                  heloName: toOptionalValue(e.target.value.trim()),
                })
              }
              size="small"
              className="w-full max-w-[250px]"
              placeholder={t('notifiers.email.heloNamePlaceholder')}
            />
          </AdvancedFieldRowComponent>

          <AdvancedFieldRowComponent
            label={t('notifiers.fields.skipTlsVerify')}
            tooltip={t('notifiers.email.skipTlsTooltip')}
          >
            <Checkbox
              checked={settings.isInsecureSkipVerify}
              onChange={(e) =>
                onChange({
                  ...settings,
                  isInsecureSkipVerify: e.target.checked,
                })
              }
            >
              {t('notifiers.fields.skipTls')}
            </Checkbox>
          </AdvancedFieldRowComponent>
        </>
      )}
    </>
  );
}
