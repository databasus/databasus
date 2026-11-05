import type { ReactNode } from 'react';

interface Props {
  label: string;
  controlId?: string;
  isTopAligned?: boolean;
  children: ReactNode;
}

export const ConnectionFieldRow = ({ label, controlId, isTopAligned, children }: Props) => (
  <div
    className={`mb-1 flex w-full flex-col items-start sm:flex-row ${isTopAligned ? 'sm:items-start' : 'sm:items-center'}`}
  >
    {controlId ? (
      <label htmlFor={controlId} className="mb-1 min-w-0 sm:mb-0 sm:w-[150px] sm:shrink-0 sm:pr-2">
        {label}
      </label>
    ) : (
      <div className="mb-1 min-w-0 sm:mb-0 sm:w-[150px] sm:shrink-0 sm:pr-2">{label}</div>
    )}
    {children}
  </div>
);
