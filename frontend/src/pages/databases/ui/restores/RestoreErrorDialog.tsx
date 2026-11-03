import { CopyOutlined } from '@ant-design/icons';
import { Alert, App, Button, Modal } from 'antd';
import { Trans, useTranslation } from 'react-i18next';

import type { Restore } from '../../../../entities/restores';
import { BrowserClipboard } from '../../../../shared/lib/BrowserClipboard';

interface Props {
  restore: Restore;
  onClose: () => void;
}

export const RestoreErrorDialog = ({ restore, onClose }: Props) => {
  const { t } = useTranslation();
  const { message } = App.useApp();

  return (
    <Modal
      title={t('restores.errorDetails.title')}
      open
      onCancel={onClose}
      maskClosable={false}
      footer={
        <Button
          icon={<CopyOutlined />}
          onClick={() => {
            BrowserClipboard.copyToClipboard(restore.failMessage || '');
            message.success(t('restores.errorDetails.copied'));
          }}
        >
          {t('common.actions.copy')}
        </Button>
      }
    >
      {restore.failMessage?.includes('must be owner of extension') && (
        <Alert
          className="mb-4"
          type="warning"
          message={
            <Trans
              i18nKey="restores.errorDetails.excludeExtensionsTip"
              components={{ bold: <strong /> }}
            />
          }
        />
      )}
      <div className="overflow-y-auto text-sm whitespace-pre-wrap" style={{ height: '400px' }}>
        {restore.failMessage}
      </div>
    </Modal>
  );
};
