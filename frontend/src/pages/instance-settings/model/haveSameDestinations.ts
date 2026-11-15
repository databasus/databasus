// The order of destinations means nothing, so only membership counts.
export const haveSameDestinations = (first: string[], second: string[]): boolean =>
  first.length === second.length && first.every((storageId) => second.includes(storageId));
