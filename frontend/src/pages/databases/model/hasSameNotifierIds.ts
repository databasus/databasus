interface NotifierIdsComparison {
  acknowledgedIds: string[];
  shownIds: string[];
}

export const hasSameNotifierIds = ({ acknowledgedIds, shownIds }: NotifierIdsComparison): boolean =>
  acknowledgedIds.length === shownIds.length &&
  shownIds.every((id) => acknowledgedIds.includes(id));
