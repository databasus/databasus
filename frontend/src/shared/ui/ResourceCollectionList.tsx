import { TextInput } from '@mantine/core';
import { Fragment, type ReactNode } from 'react';

interface Props<Item> {
  items: Item[];
  getKey: (item: Item) => string;
  getName: (item: Item) => string;
  renderCard: (item: Item) => ReactNode;
  addAction?: ReactNode;
  description: string;
  searchLabel: string;
  searchQuery: string;
  onSearchChange: (query: string) => void;
  noSearchResults: (query: string) => string;
}

export const ResourceCollectionList = <Item,>({
  items,
  getKey,
  getName,
  renderCard,
  addAction,
  description,
  searchLabel,
  searchQuery,
  onSearchChange,
  noSearchResults,
}: Props<Item>) => {
  const hasLargeCollection = items.length >= 5;
  const filteredItems = items.filter((item) =>
    getName(item).toLowerCase().includes(searchQuery.toLowerCase()),
  );

  return (
    <>
      {hasLargeCollection && addAction}
      {(hasLargeCollection || searchQuery !== '') && (
        <TextInput
          aria-label={searchLabel}
          placeholder={searchLabel}
          value={searchQuery}
          onChange={(event) => onSearchChange(event.target.value)}
        />
      )}
      {filteredItems.map((item) => (
        <Fragment key={getKey(item)}>{renderCard(item)}</Fragment>
      ))}
      {searchQuery && filteredItems.length === 0 && (
        <p className="text-sm text-muted">{noSearchResults(searchQuery)}</p>
      )}
      {!hasLargeCollection && addAction}
      <div className="mx-3 text-center text-xs text-muted">{description}</div>
    </>
  );
};
