import { ActionIcon, FileButton, ThemeIcon, UnstyledButton } from '@mantine/core';
import { FileArchive, Upload, X } from 'lucide-react';
import { type DragEvent, useId, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { useLocale } from '@/shared/i18n';
import { formatBytes } from '@/shared/lib';

interface Props {
  archive: File | null;
  error?: string;
  isDisabled: boolean;
  onChange: (archive: File | null) => void;
}

export function RecoveryArchivePicker({ archive, error, isDisabled, onChange }: Props) {
  const { t } = useTranslation();
  const { formatNumber } = useLocale();
  const labelId = useId();
  const contentId = useId();
  const errorId = useId();
  const resetRef = useRef<() => void>(null);
  const [isDraggedOver, setIsDraggedOver] = useState(false);

  const drop = (event: DragEvent) => {
    event.preventDefault();
    setIsDraggedOver(false);

    const droppedArchive = event.dataTransfer.files[0];
    if (droppedArchive && !isDisabled) onChange(droppedArchive);
  };

  const clear = () => {
    resetRef.current?.();
    onChange(null);
  };

  return (
    <div>
      <p id={labelId} className="mb-1 font-medium">
        {t('instanceRecovery.form.archive')}
      </p>
      <div className="relative">
        <FileButton resetRef={resetRef} disabled={isDisabled} onChange={onChange}>
          {(props) => (
            <UnstyledButton
              {...props}
              disabled={isDisabled}
              aria-labelledby={`${labelId} ${contentId}`}
              aria-describedby={error ? errorId : undefined}
              className={`flex w-full items-center gap-3 rounded-md border border-dashed px-4 py-3 pe-12 text-start text-sm transition-colors hover:bg-hover disabled:cursor-not-allowed disabled:opacity-60 ${
                error ? 'border-danger' : isDraggedOver ? 'border-accent bg-hover' : 'border-border'
              }`}
              onDragOver={(event) => {
                event.preventDefault();
                setIsDraggedOver(true);
              }}
              onDragLeave={() => setIsDraggedOver(false)}
              onDrop={drop}
            >
              <ThemeIcon variant="light" color="accent" size={40} radius="xl">
                {archive ? <FileArchive aria-hidden size={20} /> : <Upload aria-hidden size={20} />}
              </ThemeIcon>
              <span id={contentId} className="min-w-0 flex-1">
                <span className="block font-medium wrap-anywhere">
                  {archive ? archive.name : t('instanceRecovery.form.archivePlaceholder')}
                </span>
                <span className="block text-xs text-muted">
                  {archive
                    ? formatBytes(archive.size, formatNumber)
                    : t('instanceRecovery.form.archiveHint')}
                </span>
              </span>
            </UnstyledButton>
          )}
        </FileButton>

        {archive && !isDisabled && (
          <ActionIcon
            className="absolute end-3 top-1/2 -translate-y-1/2"
            variant="subtle"
            aria-label={t('instanceRecovery.form.archiveClear')}
            onClick={clear}
          >
            <X aria-hidden size={16} />
          </ActionIcon>
        )}
      </div>
      {error && (
        <p id={errorId} className="mt-1 text-xs text-danger">
          {error}
        </p>
      )}
    </div>
  );
}
