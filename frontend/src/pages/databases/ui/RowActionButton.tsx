import { Button, Tooltip } from 'antd';
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
  <Tooltip title={title}>
    <Button
      type="text"
      size="small"
      className={`${className} !text-lg`}
      aria-label={title}
      icon={icon}
      loading={isPending}
      disabled={isDisabled}
      onClick={onClick}
    />
  </Tooltip>
);
