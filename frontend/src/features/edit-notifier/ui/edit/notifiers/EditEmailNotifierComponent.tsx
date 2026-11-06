import { Checkbox, Form, Input, Select } from 'antd';
import { useId, useState } from 'react';
import { useTranslation } from 'react-i18next';

import {
  EMAIL_NOTIFIER_SECURITY_LABEL_KEYS,
  EmailNotifierSecurity,
  type EmailNotifierSettings,
  getDefaultEmailNotifierSecurity,
  getSecurityAfterPortChange,
} from '@/entities/notifiers';
import { toOptionalValue } from '@/shared/lib';
import {
  AddressChangedSecretNoticeComponent,
  AdvancedFieldRowComponent,
  AdvancedSettingsToggleComponent,
} from '@/shared/ui';

interface Props {
  settings: EmailNotifierSettings;
  isSaved: boolean;
  isAddressChanged: boolean;
  onChange: (settings: EmailNotifierSettings) => void;
}

export function EditEmailNotifierComponent({
  settings,
  isSaved,
  isAddressChanged,
  onChange,
}: Props) {
  const [isShowAdvanced, setIsShowAdvanced] = useState(
    settings.isInsecureSkipVerify ||
      !!settings.heloName ||
      settings.security !== getDefaultEmailNotifierSecurity(settings.smtpPort),
  );
  const [touchedFields, setTouchedFields] = useState<Record<string, boolean>>({});
  const { t } = useTranslation();
  const fieldPrefix = useId();

  const isSmtpPasswordRequiredAgain =
    isAddressChanged && !!settings.smtpUser && !settings.smtpPassword;
  const invalidFields = {
    isTargetEmailInvalid: !/^.+@[^\s@]+$/.test(settings.targetEmail ?? ''),
    isSmtpHostInvalid: !settings.smtpHost?.trim(),
    isSmtpPortInvalid:
      !Number.isInteger(settings.smtpPort) || settings.smtpPort < 1 || settings.smtpPort > 65535,
  };
  const fieldErrors = {
    targetEmail:
      touchedFields.targetEmail && invalidFields.isTargetEmailInvalid
        ? t('notifiers.email.validation.targetEmail')
        : undefined,
    smtpHost:
      touchedFields.smtpHost && invalidFields.isSmtpHostInvalid
        ? t('notifiers.email.validation.smtpHost')
        : undefined,
    smtpPort:
      touchedFields.smtpPort && invalidFields.isSmtpPortInvalid
        ? t('notifiers.email.validation.smtpPort')
        : undefined,
  };

  return (
    <div className="min-w-0">
      <Form.Item
        layout="vertical"
        label={t('notifiers.email.targetEmail')}
        htmlFor={`${fieldPrefix}-targetEmail`}
        validateStatus={fieldErrors.targetEmail ? 'error' : undefined}
        help={
          fieldErrors.targetEmail ? (
            <span className="text-danger" id={`${fieldPrefix}-targetEmail-error`}>
              {fieldErrors.targetEmail}
            </span>
          ) : undefined
        }
        extra={
          <span id={`${fieldPrefix}-targetEmail-help`}>
            {t('notifiers.email.targetEmailTooltip')}
          </span>
        }
      >
        <Input
          id={`${fieldPrefix}-targetEmail`}
          type="email"
          value={settings.targetEmail ?? ''}
          onChange={(event) => onChange({ ...settings, targetEmail: event.target.value.trim() })}
          size="small"
          className="w-full min-w-0"
          onBlur={() => setTouchedFields({ ...touchedFields, targetEmail: true })}
          aria-invalid={!!fieldErrors.targetEmail}
          aria-describedby={
            [
              `${fieldPrefix}-targetEmail-help`,
              fieldErrors.targetEmail ? `${fieldPrefix}-targetEmail-error` : undefined,
            ]
              .filter(Boolean)
              .join(' ') || undefined
          }
          placeholder="example@gmail.com"
        />
      </Form.Item>

      <Form.Item
        layout="vertical"
        label={t('notifiers.email.smtpHost')}
        htmlFor={`${fieldPrefix}-smtpHost`}
        validateStatus={fieldErrors.smtpHost ? 'error' : undefined}
        help={
          fieldErrors.smtpHost ? (
            <span className="text-danger" id={`${fieldPrefix}-smtpHost-error`}>
              {fieldErrors.smtpHost}
            </span>
          ) : undefined
        }
      >
        <Input
          id={`${fieldPrefix}-smtpHost`}
          type="text"
          value={settings.smtpHost ?? ''}
          onChange={(event) => onChange({ ...settings, smtpHost: event.target.value.trim() })}
          size="small"
          className="w-full min-w-0"
          onBlur={() => setTouchedFields({ ...touchedFields, smtpHost: true })}
          aria-invalid={!!fieldErrors.smtpHost}
          aria-describedby={
            [fieldErrors.smtpHost ? `${fieldPrefix}-smtpHost-error` : undefined]
              .filter(Boolean)
              .join(' ') || undefined
          }
          // eslint-disable-next-line i18next/no-literal-string -- example hostname
          placeholder="smtp.gmail.com"
        />
      </Form.Item>

      <Form.Item
        layout="vertical"
        label={t('notifiers.email.smtpPort')}
        htmlFor={`${fieldPrefix}-smtpPort`}
        validateStatus={fieldErrors.smtpPort ? 'error' : undefined}
        help={
          fieldErrors.smtpPort ? (
            <span className="text-danger" id={`${fieldPrefix}-smtpPort-error`}>
              {fieldErrors.smtpPort}
            </span>
          ) : undefined
        }
      >
        <Input
          id={`${fieldPrefix}-smtpPort`}
          type="number"
          value={settings.smtpPort ?? ''}
          onChange={(event) => {
            const nextPort = Number(event.target.value);
            onChange({
              ...settings,
              smtpPort: nextPort,
              security: getSecurityAfterPortChange(settings.security, settings.smtpPort, nextPort),
            });
          }}
          size="small"
          className="w-full min-w-0"
          onBlur={() => setTouchedFields({ ...touchedFields, smtpPort: true })}
          aria-invalid={!!fieldErrors.smtpPort}
          aria-describedby={
            [fieldErrors.smtpPort ? `${fieldPrefix}-smtpPort-error` : undefined]
              .filter(Boolean)
              .join(' ') || undefined
          }
          placeholder="25"
          min={1}
          max={65535}
        />
      </Form.Item>

      <Form.Item
        layout="vertical"
        label={t('notifiers.email.smtpUser')}
        htmlFor={`${fieldPrefix}-smtpUser`}
      >
        <Input
          id={`${fieldPrefix}-smtpUser`}
          type="text"
          value={settings.smtpUser ?? ''}
          onChange={(event) =>
            onChange({ ...settings, smtpUser: toOptionalValue(event.target.value.trim()) })
          }
          size="small"
          className="w-full min-w-0"
          placeholder="user@gmail.com"
        />
      </Form.Item>

      <Form.Item
        layout="vertical"
        label={t('notifiers.email.smtpPassword')}
        htmlFor={`${fieldPrefix}-smtpPassword`}
      >
        <Input
          id={`${fieldPrefix}-smtpPassword`}
          type="password"
          value={settings.smtpPassword ?? ''}
          onChange={(event) =>
            onChange({ ...settings, smtpPassword: toOptionalValue(event.target.value) })
          }
          size="small"
          className="w-full min-w-0"
          aria-describedby={
            isSmtpPasswordRequiredAgain ? `${fieldPrefix}-smtpPassword-notice` : undefined
          }
          placeholder={
            isSaved && !isAddressChanged
              ? t('notifiers.secretKeptPlaceholder')
              : t('notifiers.email.smtpPasswordPlaceholder')
          }
        />
        {isSmtpPasswordRequiredAgain && (
          <AddressChangedSecretNoticeComponent id={`${fieldPrefix}-smtpPassword-notice`} />
        )}
      </Form.Item>

      <Form.Item
        layout="vertical"
        label={t('notifiers.email.from')}
        htmlFor={`${fieldPrefix}-from`}
        extra={<span id={`${fieldPrefix}-from-help`}>{t('notifiers.email.fromTooltip')}</span>}
      >
        <Input
          id={`${fieldPrefix}-from`}
          type="text"
          value={settings.from ?? ''}
          onChange={(event) =>
            onChange({ ...settings, from: toOptionalValue(event.target.value.trim()) })
          }
          size="small"
          className="w-full min-w-0"
          aria-describedby={[`${fieldPrefix}-from-help`].filter(Boolean).join(' ') || undefined}
          placeholder="example@example.com"
        />
      </Form.Item>

      <AdvancedSettingsToggleComponent
        regionId={`${fieldPrefix}-advanced`}
        label={t('notifiers.advancedSettings')}
        isShowAdvanced={isShowAdvanced}
        onToggle={() => setIsShowAdvanced(!isShowAdvanced)}
      />

      {isShowAdvanced && (
        <div id={`${fieldPrefix}-advanced`}>
          <AdvancedFieldRowComponent
            controlId={`${fieldPrefix}-security`}
            label={t('notifiers.email.security')}
            tooltip={t('notifiers.email.securityTooltip')}
          >
            <Select
              id={`${fieldPrefix}-security`}
              aria-describedby={`${fieldPrefix}-security-help`}
              value={settings.security}
              onChange={(security: EmailNotifierSecurity) => onChange({ ...settings, security })}
              size="small"
              className="w-full min-w-0 sm:w-[250px]"
              options={Object.values(EmailNotifierSecurity).map(
                (security: EmailNotifierSecurity) => ({
                  value: security,
                  label: t(EMAIL_NOTIFIER_SECURITY_LABEL_KEYS[security]),
                }),
              )}
            />
          </AdvancedFieldRowComponent>

          <AdvancedFieldRowComponent
            controlId={`${fieldPrefix}-heloName`}
            label={t('notifiers.email.heloName')}
            tooltip={t('notifiers.email.heloNameTooltip')}
          >
            <Input
              id={`${fieldPrefix}-heloName`}
              aria-describedby={`${fieldPrefix}-heloName-help`}
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
            controlId={`${fieldPrefix}-skipTlsVerify`}
            label={t('notifiers.fields.skipTlsVerify')}
            tooltip={t('notifiers.email.skipTlsTooltip')}
          >
            <Checkbox
              id={`${fieldPrefix}-skipTlsVerify`}
              aria-describedby={`${fieldPrefix}-skipTlsVerify-help`}
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
        </div>
      )}
    </div>
  );
}
