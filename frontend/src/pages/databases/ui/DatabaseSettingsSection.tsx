import type { ReactNode } from 'react';

interface Props {
  title: string;
  isEditable: boolean;
  onEdit: () => void;
  children: ReactNode;
}

export const DatabaseSettingsSection = ({ title, isEditable, onEdit, children }: Props) => (
  <div className="w-full lg:w-[400px]">
    <div className="mt-5 flex items-center font-bold">
      <div>{title}</div>

      {isEditable && (
        <div className="ml-2 h-4 w-4 cursor-pointer" onClick={onEdit}>
          <img src="/icons/pen-gray.svg" />
        </div>
      )}
    </div>

    <div className="mt-1 text-sm">{children}</div>
  </div>
);
