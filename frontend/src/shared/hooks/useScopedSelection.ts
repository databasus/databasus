import { useEffect, useEffectEvent, useState } from 'react';

interface Options {
  scopeKey: string;
  itemIds: string[];
  isReady: boolean;
  isMobile: boolean;
}
interface Selection {
  scopeKey: string;
  selectedItemId?: string;
  hasChosen: boolean;
}
const readSelection = (scopeKey: string): Selection => {
  try {
    return {
      scopeKey,
      selectedItemId: localStorage.getItem(scopeKey) ?? undefined,
      hasChosen: false,
    };
  } catch {
    return { scopeKey, hasChosen: false };
  }
};

export function useScopedSelection({ scopeKey, itemIds, isReady, isMobile }: Options) {
  const [selection, setSelection] = useState(() => readSelection(scopeKey));
  const currentSelection = selection.scopeKey === scopeKey ? selection : readSelection(scopeKey);

  if (selection.scopeKey !== scopeKey) setSelection(currentSelection);

  const selectedItemId = !isReady
    ? undefined
    : !currentSelection.hasChosen && isMobile
      ? undefined
      : currentSelection.selectedItemId && itemIds.includes(currentSelection.selectedItemId)
        ? currentSelection.selectedItemId
        : currentSelection.hasChosen && !currentSelection.selectedItemId
          ? undefined
          : itemIds[0];

  const select = (selectedItemId?: string) =>
    setSelection({ scopeKey, selectedItemId, hasChosen: true });
  const remove = (removedId: string) => select(itemIds.find((itemId) => itemId !== removedId));

  const acknowledgeSelection = useEffectEvent(() => {
    if (selection.selectedItemId !== selectedItemId || !selection.hasChosen)
      setSelection({ scopeKey, selectedItemId, hasChosen: true });
  });

  useEffect(() => {
    if (!isReady) return;

    acknowledgeSelection();

    try {
      if (selectedItemId) localStorage.setItem(scopeKey, selectedItemId);
      else localStorage.removeItem(scopeKey);
    } catch {}
  }, [scopeKey, selectedItemId, isReady]);

  return { selectedItemId, select, remove };
}
