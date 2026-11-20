import { ActionIcon } from '@mantine/core';
import { Download } from 'lucide-react';
import { useState } from 'react';
import { useTranslation } from 'react-i18next';

import { type InstanceBackup, instanceBackupApi } from '@/entities/instance-backups';
import { useSessionMutation } from '@/shared/api';
import { useSessionGuard } from '@/shared/auth';
import { translateApiError } from '@/shared/i18n';
import { followFileLink } from '@/shared/lib';
import { useFeedback } from '@/shared/lib/feedback';

interface Props {
  backup: InstanceBackup;
  label: string;
}

// Root admin does not choose a storage: the copies are tried in order until one issues a download.
export function InstanceBackupDownloadAction({ backup, label }: Props) {
  const { i18n } = useTranslation();
  const notify = useFeedback();
  const isCurrent = useSessionGuard();
  const [isDownloading, setIsDownloading] = useState(false);
  const downloadableCopies = backup.copies.filter((copy) => copy.isDownloadable);

  const downloadMutation = useSessionMutation({
    mutationFn: (copyId: string) => instanceBackupApi.createDownloadToken(backup.id, copyId),
  });

  const download = async () => {
    if (isDownloading) return;

    setIsDownloading(true);
    let lastError: unknown;

    try {
      for (const copy of downloadableCopies) {
        try {
          const token = await downloadMutation.mutateAsync(copy.id);
          if (isCurrent())
            followFileLink(instanceBackupApi.getDownloadUrl(token.token), token.fileName);

          return;
        } catch (error: unknown) {
          if (!isCurrent()) return;

          lastError = error;
        }
      }

      notify({ kind: 'error', title: translateApiError(lastError, i18n.t) });
    } finally {
      if (isCurrent()) {
        setIsDownloading(false);
        downloadMutation.reset();
      }
    }
  };

  if (downloadableCopies.length === 0) return null;

  return (
    <ActionIcon
      size="md"
      variant="subtle"
      loading={isDownloading}
      aria-label={label}
      onClick={() => void download()}
    >
      <Download aria-hidden size={16} />
    </ActionIcon>
  );
}
