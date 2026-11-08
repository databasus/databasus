import { Tooltip } from '@mantine/core';
import { Info } from 'lucide-react';

interface Props {
  title: string;
}

export const ConnectionFieldTooltip = ({ title }: Props) => (
  <Tooltip className="cursor-pointer" label={title}>
    <span className="ml-2 inline-flex shrink-0 items-center align-middle text-muted" tabIndex={0}>
      <Info aria-label={title} size={14} className="block" />
    </span>
  </Tooltip>
);
