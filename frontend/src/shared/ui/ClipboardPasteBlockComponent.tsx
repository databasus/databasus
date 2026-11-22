import { Button, Divider } from '@mantine/core';
import { ClipboardPaste } from 'lucide-react';

import { ClipboardPasteModalComponent } from '@/shared/ui/ClipboardPasteModalComponent';

interface Props {
  title: string;
  hint: string;
  example: string;
  pasteLabel: string;
  dividerLabel: string;
  onPaste: () => void;
  pasteModal: {
    open: boolean;
    onSubmit(text: string): void;
    onCancel(): void;
  };
}

// The main way to fill a form from text a provider's console hands out, above the individual fields.
export const ClipboardPasteBlockComponent = ({
  title,
  hint,
  example,
  pasteLabel,
  dividerLabel,
  onPaste,
  pasteModal,
}: Props) => (
  <>
    <div>
      <div className="font-semibold">{title}</div>

      <p className="mt-1 mb-2 text-xs leading-snug text-muted">{hint}</p>

      <pre className="rounded-md bg-code p-2 font-mono text-xs break-all whitespace-pre-wrap text-muted">
        <code>{example}</code>
      </pre>

      <Button
        className="mt-2"
        variant="light"
        size="xs"
        leftSection={<ClipboardPaste aria-hidden size={14} />}
        onClick={onPaste}
      >
        {pasteLabel}
      </Button>

      <ClipboardPasteModalComponent {...pasteModal} />
    </div>

    <Divider my="md" labelPosition="center" label={dividerLabel} />
  </>
);
