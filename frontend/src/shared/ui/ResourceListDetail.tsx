import type { ReactNode, Ref } from 'react';

interface Props {
  list: ReactNode;
  detail?: ReactNode;
  backAction?: ReactNode;
  isMobile: boolean;
  hasSelection: boolean;
  detailScrollRef?: Ref<HTMLDivElement>;
}

export function ResourceListDetail({
  list,
  detail,
  backAction,
  isMobile,
  hasSelection,
  detailScrollRef,
}: Props) {
  return (
    <div className="flex min-h-0 min-w-0 flex-1">
      {(!isMobile || !hasSelection) && (
        <div className="min-h-0 w-full shrink-0 overflow-y-auto md:mr-3 md:w-62.5">{list}</div>
      )}
      {hasSelection && (
        <div className="flex min-h-0 min-w-0 flex-1 flex-col">
          {isMobile && <div className="mb-2 shrink-0">{backAction}</div>}
          <div
            ref={detailScrollRef}
            className="min-h-0 min-w-0 flex-1 overflow-y-auto wrap-anywhere"
          >
            {detail}
          </div>
        </div>
      )}
    </div>
  );
}
