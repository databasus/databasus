import { Checkbox, TextInput } from '@mantine/core';
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
import { WrappingSelect } from '@/shared/ui';
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
      <div className="mb-3 min-w-0">
        <label htmlFor={`${fieldPrefix}-targetEmail`} className="mb-1 block text-sm text-content">
          {t('notifiers.email.targetEmail')}
        </label>
        <TextInput
          id={`${fieldPrefix}-targetEmail`}
          description={t('notifiers.email.targetEmailTooltip')}
          type="email"
          value={settings.targetEmail ?? ''}
          onChange={(event) => onChange({ ...settings, targetEmail: event.target.value.trim() })}
          size="xs"
          className="w-full min-w-0"
          onBlur={() => setTouchedFields({ ...touchedFields, targetEmail: true })}
          aria-invalid={!!fieldErrors.targetEmail}
          placeholder="example@gmail.com"
          error={
            fieldErrors.targetEmail ? (
              <span className="text-danger">{fieldErrors.targetEmail}</span>
            ) : undefined
          }
        />
      </div>

      <div className="mb-3 min-w-0">
        <label htmlFor={`${fieldPrefix}-smtpHost`} className="mb-1 block text-sm text-content">
          {t('notifiers.email.smtpHost')}
        </label>
        <TextInput
          id={`${fieldPrefix}-smtpHost`}
          type="text"
          value={settings.smtpHost ?? ''}
          onChange={(event) => onChange({ ...settings, smtpHost: event.target.value.trim() })}
          size="xs"
          className="w-full min-w-0"
          onBlur={() => setTouchedFields({ ...touchedFields, smtpHost: true })}
          aria-invalid={!!fieldErrors.smtpHost}
          // eslint-disable-next-line i18next/no-literal-string -- example hostname
          placeholder="smtp.gmail.com"
          error={
            fieldErrors.smtpHost ? (
              <span className="text-danger">{fieldErrors.smtpHost}</span>
            ) : undefined
          }
        />
      </div>

      <div className="mb-3 min-w-0">
        <label htmlFor={`${fieldPrefix}-smtpPort`} className="mb-1 block text-sm text-content">
          {t('notifiers.email.smtpPort')}
        </label>
        <TextInput
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
          size="xs"
          className="w-full min-w-0"
          onBlur={() => setTouchedFields({ ...touchedFields, smtpPort: true })}
          aria-invalid={!!fieldErrors.smtpPort}
          placeholder="25"
          min={1}
          max={65535}
          error={
            fieldErrors.smtpPort ? (
              <span className="text-danger">{fieldErrors.smtpPort}</span>
            ) : undefined
          }
        />
      </div>

      <div className="mb-3 min-w-0">
        <label htmlFor={`${fieldPrefix}-smtpUser`} className="mb-1 block text-sm text-content">
          {t('notifiers.email.smtpUser')}
        </label>
        <TextInput
          id={`${fieldPrefix}-smtpUser`}
          type="text"
          value={settings.smtpUser ?? ''}
          onChange={(event) =>
            onChange({ ...settings, smtpUser: toOptionalValue(event.target.value.trim()) })
          }
          size="xs"
          className="w-full min-w-0"
          placeholder="user@gmail.com"
        />
      </div>

      <div className="mb-3 min-w-0">
        <label htmlFor={`${fieldPrefix}-smtpPassword`} className="mb-1 block text-sm text-content">
          {t('notifiers.email.smtpPassword')}
        </label>
        <TextInput
          id={`${fieldPrefix}-smtpPassword`}
          type="password"
          value={settings.smtpPassword ?? ''}
          onChange={(event) =>
            onChange({ ...settings, smtpPassword: toOptionalValue(event.target.value) })
          }
          size="xs"
          className="w-full min-w-0"
          placeholder={
            isSaved && !isAddressChanged
              ? t('notifiers.secretKeptPlaceholder')
              : t('notifiers.email.smtpPasswordPlaceholder')
          }
          description={
            isSmtpPasswordRequiredAgain && (
              <AddressChangedSecretNoticeComponent id={`${fieldPrefix}-smtpPassword-notice`} />
            )
          }
        />
      </div>

      <div className="mb-3 min-w-0">
        <label htmlFor={`${fieldPrefix}-from`} className="mb-1 block text-sm text-content">
          {t('notifiers.email.from')}
        </label>
        <TextInput
          id={`${fieldPrefix}-from`}
          description={t('notifiers.email.fromTooltip')}
          type="text"
          value={settings.from ?? ''}
          onChange={(event) =>
            onChange({ ...settings, from: toOptionalValue(event.target.value.trim()) })
          }
          size="xs"
          className="w-full min-w-0"
          aria-describedby={[`${fieldPrefix}-from-help`].filter(Boolean).join(' ') || undefined}
          placeholder="example@example.com"
        />
      </div>

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
            <WrappingSelect
              id={`${fieldPrefix}-security`}
              aria-describedby={`${fieldPrefix}-security-help`}
              value={settings.security}
              onChange={(security: EmailNotifierSecurity) => onChange({ ...settings, security })}
              size="xs"
              className="w-full min-w-0 sm:w-[250px]"
              data={Object.values(EmailNotifierSecurity).map((security: EmailNotifierSecurity) => ({
                value: security,
                label: t(EMAIL_NOTIFIER_SECURITY_LABEL_KEYS[security]),
              }))}
            />
          </AdvancedFieldRowComponent>

          <AdvancedFieldRowComponent
            controlId={`${fieldPrefix}-heloName`}
            label={t('notifiers.email.heloName')}
            tooltip={t('notifiers.email.heloNameTooltip')}
          >
            <TextInput
              id={`${fieldPrefix}-heloName`}
              description={t('notifiers.email.heloNameTooltip')}
              descriptionProps={{ className: 'sr-only' }}
              classNames={{ wrapper: 'mb-0' }}
              aria-describedby={`${fieldPrefix}-heloName-help`}
              value={settings.heloName ?? ''}
              onChange={(e) =>
                onChange({
                  ...settings,
                  heloName: toOptionalValue(e.target.value.trim()),
                })
              }
              size="xs"
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
              classNames={{ body: 'items-center' }}
              aria-describedby={`${fieldPrefix}-skipTlsVerify-help`}
              checked={settings.isInsecureSkipVerify}
              onChange={(e) =>
                onChange({
                  ...settings,
                  isInsecureSkipVerify: e.target.checked,
                })
              }
              label={<>{t('notifiers.fields.skipTls')}</>}
            />
          </AdvancedFieldRowComponent>
        </div>
      )}
    </div>
  );
}
