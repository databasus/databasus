const VISIBLE_EDGE_LENGTH = 4;
const MASK = '••••••••••••';

// A fixed mask hides the key's length as well as its middle.
export const getMasterKeyPreview = (key: string): string => {
  const trimmedKey = key.trim();
  if (trimmedKey.length <= VISIBLE_EDGE_LENGTH * 2) return MASK;

  return `${trimmedKey.slice(0, VISIBLE_EDGE_LENGTH)}${MASK}${trimmedKey.slice(-VISIBLE_EDGE_LENGTH)}`;
};
