import { InfoCircleOutlined } from '@ant-design/icons';
import { Tooltip } from 'antd';
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

      <Tooltip className="cursor-pointer" title={tooltip}>
        <span id={`${controlId}-help`} className="ml-2 text-muted" tabIndex={0}>
          <InfoCircleOutlined aria-label={tooltip} />
        </span>
      </Tooltip>
    </div>
  </div>
);
