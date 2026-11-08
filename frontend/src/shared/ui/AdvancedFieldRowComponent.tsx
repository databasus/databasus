import { Tooltip } from '@mantine/core';
import { Info } from 'lucide-react';
import type { ReactNode } from 'react';

interface Props {
  label: string;
  tooltip: string;
  children: ReactNode;
  controlId: string;
}

export const AdvancedFieldRowComponent = ({ label, tooltip, children, controlId }: Props) => (
  <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
    <label htmlFor={controlId} className="mb-1 min-w-0 sm:mb-0 sm:w-[150px] sm:shrink-0 sm:pr-2">
      {label}
    </label>
    <div className="flex min-w-0 items-center">
      {children}

      <Tooltip label={tooltip} events={{ hover: true, focus: true, touch: false }}>
        <span
          id={`${controlId}-help`}
          className="ml-2 inline-flex shrink-0 items-center text-muted"
          tabIndex={0}
        >
          <Info size="1em" aria-label={tooltip} className="block" />
        </span>
      </Tooltip>
    </div>
  </div>
);
