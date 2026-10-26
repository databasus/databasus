import { InfoCircleOutlined } from '@ant-design/icons';
import { Input, InputNumber, Switch, Tooltip } from 'antd';
import { useTranslation } from 'react-i18next';

import type { NasStorageSettings } from '../../../../../entity/storages';
import { toOptionalValue } from '../../../../../shared/lib';

interface Props {
  settings: NasStorageSettings;
  onChange: (settings: NasStorageSettings) => void;
}

export function EditNASStorageComponent({ settings, onChange }: Props) {
  const { t } = useTranslation();
  const shareHasSlash = settings.share?.includes('/') || settings.share?.includes('\\');

  return (
    <>
      <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
        <div className="mb-1 min-w-[110px] sm:mb-0 sm:pr-2">{t('common.fields.host')}</div>
        <Input
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
            <div className="mt-1 max-w-[250px] text-xs text-yellow-600">
              {t('storages.edit.nas.shareWithSlashWarning')}
            </div>
          )}
        </div>
      </div>

      <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
        <div className="mb-1 min-w-[110px] sm:mb-0 sm:pr-2">{t('common.fields.username')}</div>
        <Input
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
        <Input.Password
          value={settings.password || ''}
          onChange={(e) => {
            onChange({ ...settings, password: toOptionalValue(e.target.value) });
          }}
          size="small"
          className="w-full max-w-[250px]"
          placeholder={t('storages.edit.passwordPlaceholder')}
          autoComplete="off"
          data-1p-ignore
          data-lpignore="true"
          data-form-type="other"
        />
      </div>

      <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
        <div className="mb-1 min-w-[110px] sm:mb-0 sm:pr-2">{t('storages.fields.useSsl')}</div>
        <div className="flex items-center">
          <Switch
            checked={settings.isSsl || false}
            onChange={(checked) => {
              onChange({ ...settings, isSsl: checked });
            }}
            size="small"
          />

          <Tooltip className="cursor-pointer" title={t('storages.edit.nas.useSslTooltip')}>
            <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
          </Tooltip>
        </div>
      </div>

      <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
        <div className="mb-1 min-w-[110px] sm:mb-0 sm:pr-2">{t('storages.fields.domain')}</div>
        <div className="flex items-center">
          <Input
            value={settings.domain || ''}
            onChange={(e) => {
              onChange({ ...settings, domain: e.target.value.trim() || undefined });
            }}
            size="small"
            className="w-full max-w-[250px]"
            placeholder={t('storages.edit.optionalPlaceholder', { example: 'WORKGROUP' })}
          />

          <Tooltip className="cursor-pointer" title={t('storages.edit.nas.domainTooltip')}>
            <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
          </Tooltip>
        </div>
      </div>

      <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
        <div className="mb-1 min-w-[110px] sm:mb-0 sm:pr-2">{t('storages.fields.path')}</div>
        <div className="flex items-center">
          <Input
            value={settings.path || ''}
            onChange={(e) => {
              let pathValue = e.target.value.trim();
              // Remove leading slash if present
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
            <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
          </Tooltip>
        </div>
      </div>
    </>
  );
}
