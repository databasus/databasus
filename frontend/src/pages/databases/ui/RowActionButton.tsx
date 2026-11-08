import { ActionIcon, Tooltip } from '@mantine/core';
import type { ReactNode } from 'react';

interface Props {
  title: string;
  icon: ReactNode;
  className: string;
  isPending?: boolean;
  isDisabled?: boolean;
  onClick: () => void;
}

export const RowActionButton = ({
  title,
  icon,
  className,
  isPending,
  isDisabled,
  onClick,
}: Props) => (
  <Tooltip label={title}>
    <ActionIcon
      variant="subtle"
      size="sm"
      className={`${className} !text-lg`}
      aria-label={title}
      loading={isPending}
      disabled={isDisabled}
      onClick={onClick}
      aria-busy={isPending}
    >
      {icon}
    </ActionIcon>
  </Tooltip>
);
