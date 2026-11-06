import { type ReactNode, useId } from 'react';

import { EditIconButton } from '@/pages/databases/ui/EditIconButton';

interface Props {
  title: string;
  isEditable: boolean;
  onEdit: () => void;
  children: ReactNode;
}

export const DatabaseSettingsSection = ({ title, isEditable, onEdit, children }: Props) => {
  const titleId = useId();

  return (
    <div className="w-full lg:w-[400px]">
      <div className="mt-5 flex items-center font-bold">
        <div id={titleId}>{title}</div>

        {isEditable && (
          <EditIconButton subjectId={titleId} className="ml-2 !h-4 !w-4" onClick={onEdit} />
        )}
      </div>

      <div className="mt-1 text-sm">{children}</div>
    </div>
  );
};
