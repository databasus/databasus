import { InfoCircleOutlined } from '@ant-design/icons';
import { Input, InputNumber, Switch, Tooltip } from 'antd';
import { useId } from 'react';
import { useTranslation } from 'react-i18next';

import type { NasStorageSettings } from '@/entities/storages';
import { toOptionalValue } from '@/shared/lib';
import { AddressChangedSecretNoticeComponent } from '@/shared/ui';

interface Props {
  settings: NasStorageSettings;
  isAddressChanged: boolean;
  onChange: (settings: NasStorageSettings) => void;
}

export function EditNASStorageComponent({ settings, isAddressChanged, onChange }: Props) {
  const { t } = useTranslation();
  const passwordNoticeId = useId();
  const isPasswordRequiredAgain = isAddressChanged && !settings.password;
  const shareHasSlash = settings.share?.includes('/') || settings.share?.includes('\\');

  return (
    <>
      <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
        <div className="mb-1 min-w-[110px] sm:mb-0 sm:pr-2">{t('common.fields.host')}</div>
        <Input
          aria-label={t('common.fields.host')}
          value={settings.host || ''}
          onChange={(e) => {
            onChange({ ...settings, host: e.target.value.trim() });
          }}
          size="small"
          className="w-full max-w-[250px]"
          placeholder="192.168.1.100"
        />
      </div>

      <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
        <div className="mb-1 min-w-[110px] sm:mb-0 sm:pr-2">{t('common.fields.port')}</div>
        <InputNumber
          aria-label={t('common.fields.port')}
          value={settings.port}
          onChange={(value) => {
            if (!value) return;
            onChange({ ...settings, port: value });
          }}
          size="small"
          className="w-full max-w-[250px]"
          min={1}
          max={65535}
          placeholder="445"
        />
      </div>

      <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
        <div className="mb-1 min-w-[110px] sm:mb-0 sm:pr-2">{t('storages.fields.share')}</div>
        <div className="flex flex-col">
          <Input
            aria-label={t('storages.fields.share')}
            value={settings.share || ''}
            onChange={(e) => {
              onChange({ ...settings, share: e.target.value.trim() });
            }}
            size="small"
            className="w-full max-w-[250px]"
            // eslint-disable-next-line i18next/no-literal-string -- example share name
            placeholder="shared_folder"
            status={shareHasSlash ? 'warning' : undefined}
          />
          {shareHasSlash && (
            <div className="mt-1 max-w-[250px] text-xs text-warning">
              {t('storages.edit.nas.shareWithSlashWarning')}
            </div>
          )}
        </div>
      </div>

      <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
        <div className="mb-1 min-w-[110px] sm:mb-0 sm:pr-2">{t('common.fields.username')}</div>
        <Input
          aria-label={t('common.fields.username')}
          value={settings.username || ''}
          onChange={(e) => {
            onChange({ ...settings, username: e.target.value.trim() });
          }}
          size="small"
          className="w-full max-w-[250px]"
          placeholder={t('storages.edit.usernamePlaceholder')}
        />
      </div>

      <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
        <div className="mb-1 min-w-[110px] sm:mb-0 sm:pr-2">{t('common.fields.password')}</div>
        <div className="w-full max-w-[250px]">
          <Input.Password
            aria-label={t('common.fields.password')}
            aria-describedby={isPasswordRequiredAgain ? passwordNoticeId : undefined}
            value={settings.password || ''}
            onChange={(e) => {
              onChange({ ...settings, password: toOptionalValue(e.target.value) });
            }}
            size="small"
            className="w-full"
            placeholder={t('storages.edit.passwordPlaceholder')}
            autoComplete="off"
            data-1p-ignore
            data-lpignore="true"
            data-form-type="other"
          />
          {isPasswordRequiredAgain && <AddressChangedSecretNoticeComponent id={passwordNoticeId} />}
        </div>
      </div>

      <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
        <div className="mb-1 min-w-[110px] sm:mb-0 sm:pr-2">{t('storages.fields.useSsl')}</div>
        <div className="flex items-center">
          <Switch
            aria-label={t('storages.fields.useSsl')}
            checked={settings.isSsl || false}
            onChange={(checked) => {
              onChange({ ...settings, isSsl: checked });
            }}
            size="small"
          />

          <Tooltip className="cursor-pointer" title={t('storages.edit.nas.useSslTooltip')}>
            <InfoCircleOutlined className="ml-2" />
          </Tooltip>
        </div>
      </div>

      <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
        <div className="mb-1 min-w-[110px] sm:mb-0 sm:pr-2">{t('storages.fields.domain')}</div>
        <div className="flex items-center">
          <Input
            aria-label={t('storages.fields.domain')}
            value={settings.domain || ''}
            onChange={(e) => {
              onChange({ ...settings, domain: e.target.value.trim() || undefined });
            }}
            size="small"
            className="w-full max-w-[250px]"
            placeholder={t('storages.edit.optionalPlaceholder', { example: 'WORKGROUP' })}
          />

          <Tooltip className="cursor-pointer" title={t('storages.edit.nas.domainTooltip')}>
            <InfoCircleOutlined className="ml-2" />
          </Tooltip>
        </div>
      </div>

      <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
        <div className="mb-1 min-w-[110px] sm:mb-0 sm:pr-2">{t('storages.fields.path')}</div>
        <div className="flex items-center">
          <Input
            aria-label={t('storages.fields.path')}
            value={settings.path || ''}
            onChange={(e) => {
              let pathValue = e.target.value.trim();

              if (pathValue.startsWith('/')) {
                pathValue = pathValue.substring(1);
              }

              onChange({ ...settings, path: pathValue || undefined });
            }}
            size="small"
            className="w-full max-w-[250px]"
            placeholder={t('storages.edit.nas.pathPlaceholder', { example: 'backups' })}
          />

          <Tooltip className="cursor-pointer" title={t('storages.edit.nas.pathTooltip')}>
            <InfoCircleOutlined className="ml-2" />
          </Tooltip>
        </div>
      </div>
    </>
  );
}
