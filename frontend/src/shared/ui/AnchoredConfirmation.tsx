import { Button, Popover } from '@mantine/core';
import {
  cloneElement,
  useId,
  useState,
  type MouseEventHandler,
  type ReactElement,
  type ReactNode,
} from 'react';
import { useTranslation } from 'react-i18next';

interface Props {
  title: ReactNode;
  description?: ReactNode;
  confirmLabel: string;
  isConfirming?: boolean;
  onConfirm: () => void | Promise<void>;
  children: ReactElement<{
    onClick?: MouseEventHandler<HTMLElement>;
  }>;
}

export const AnchoredConfirmation = ({
  title,
  description,
  confirmLabel,
  isConfirming,
  onConfirm,
  children,
}: Props) => {
  const { t } = useTranslation();
  const titleId = useId();
  const [isOpen, setIsOpen] = useState(false);
  const confirm = async () => {
    await onConfirm();
    setIsOpen(false);
  };

  return (
    <Popover
      opened={isOpen}
      onChange={setIsOpen}
      position="top-end"
      withArrow
      trapFocus
      closeOnEscape={false}
      returnFocus
    >
      <Popover.Target>
        {cloneElement(children, {
          onClick: (event) => {
            children.props.onClick?.(event);
            setIsOpen(true);
          },
        })}
      </Popover.Target>
      <Popover.Dropdown
        role="dialog"
        aria-labelledby={titleId}
        onKeyDown={(event) => {
          if (event.key !== 'Escape') return;
          event.stopPropagation();
          setIsOpen(false);
        }}
      >
        <div className="max-w-80 text-sm [overflow-wrap:anywhere]">
          <div id={titleId} className="font-semibold">
            {title}
          </div>
          {description && <div className="mt-1">{description}</div>}
          <div className="mt-3 flex flex-wrap justify-end gap-2">
            <Button size="xs" data-autofocus onClick={() => setIsOpen(false)}>
              {t('common.actions.cancel')}
            </Button>
            <Button
              size="xs"
              variant="filled"
              color="danger"
              loading={isConfirming}
              onClick={() => void confirm()}
              aria-busy={isConfirming}
            >
              {confirmLabel}
            </Button>
          </div>
        </div>
      </Popover.Dropdown>
    </Popover>
  );
};
