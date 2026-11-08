import { Alert, Button } from '@mantine/core';
import { Copy } from 'lucide-react';
import { Trans, useTranslation } from 'react-i18next';

import type { Restore } from '@/entities/restores';
import { BrowserClipboard } from '@/shared/lib/BrowserClipboard';
import { useFeedback } from '@/shared/lib/feedback';
import { Dialog } from '@/shared/ui';

interface Props {
  restore: Restore;
  onClose: () => void;
}

export const RestoreErrorDialog = ({ restore, onClose }: Props) => {
  const { t } = useTranslation();
  const notify = useFeedback();

  return (
    <Dialog
      title={t('restores.errorDetails.title')}
      opened
      onClose={onClose}
      closeOnClickOutside={false}
    >
      {restore.failMessage?.includes('must be owner of extension') && (
        <Alert
          className="mb-4"
          color="warning"
          title={
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
      <div className="mt-5 flex flex-wrap justify-end gap-2">
        <Button
          leftSection={<Copy size="1em" className="inline-block shrink-0 align-middle" />}
          onClick={() => {
            BrowserClipboard.copyToClipboard(restore.failMessage || '');
            notify({ kind: 'success', title: t('restores.errorDetails.copied') });
          }}
        >
          {t('common.actions.copy')}
        </Button>
      </div>
    </Dialog>
  );
};
