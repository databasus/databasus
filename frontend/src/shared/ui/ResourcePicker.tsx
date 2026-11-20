import { ActionIcon, Button, Combobox, Loader, Pill, PillsInput, useCombobox } from '@mantine/core';
import { Check, ChevronDown, X } from 'lucide-react';
import { useLayoutEffect, useRef, useState, type KeyboardEvent, type Ref } from 'react';
import { useTranslation } from 'react-i18next';

// A long translated label wraps inside its pill instead of overflowing a narrow picker. The pill
// keeps its content width rather than Mantine's zero basis, which a wrapping label would let
// shrink to one letter per line.
/* eslint-disable i18next/no-literal-string -- Tailwind classes, not copy */
const WRAPPING_PILL_CLASS_NAMES = {
  root: '!h-auto max-w-full !flex-none',
  label: 'wrap-anywhere !whitespace-normal',
};
/* eslint-enable i18next/no-literal-string */

interface Props {
  id?: string;
  label: string;
  data: {
    value: string;
    label: string;
  }[];
  selectedValues: string[];
  onSelect: (value: string) => void;
  onRemove?: (value: string) => void;
  onClear?: () => void;
  isSearchable?: boolean;
  isMultiple?: boolean;
  isLoading?: boolean;
  isOpen: boolean;
  onOpenChange: (isOpen: boolean) => void;
  placeholder?: string;
  searchPlaceholder?: string;
  createLabel?: string;
  onCreate?: () => void;
  targetRef?: Ref<HTMLButtonElement | HTMLInputElement>;
  className?: string;
  size?: 'xs' | 'sm';
}

export const ResourcePicker = ({
  id,
  label,
  data,
  selectedValues,
  onSelect,
  onRemove,
  onClear,
  isSearchable,
  isMultiple,
  isLoading,
  isOpen,
  onOpenChange,
  placeholder,
  searchPlaceholder,
  createLabel,
  onCreate,
  targetRef,
  className,
  size = 'xs',
}: Props) => {
  const { t } = useTranslation();
  const [query, setQuery] = useState('');
  const createButtonRef = useRef<HTMLButtonElement>(null);
  const combobox = useCombobox({
    opened: isOpen,
    onOpenedChange: onOpenChange,
    onDropdownClose: () => {
      combobox.resetSelectedOption();
      setQuery('');
    },
    onDropdownOpen: () => {
      combobox.selectActiveOption();
    },
  });

  useLayoutEffect(() => {
    if (isOpen && isSearchable) combobox.searchRef.current?.focus({ preventScroll: true });
  }, [isOpen, isSearchable, combobox.searchRef]);

  const selectedOptions = data.filter((option) => selectedValues.includes(option.value));
  const options = data.filter((option) =>
    option.label.toLocaleLowerCase().includes(query.toLocaleLowerCase()),
  );

  const handleTargetKeyDown = (event: KeyboardEvent) => {
    // Popover can close during capture before the search receives Escape.
    if (
      event.key === 'Escape' &&
      !event.nativeEvent.isComposing &&
      (isOpen || event.currentTarget === combobox.searchRef.current)
    ) {
      event.preventDefault();
      event.stopPropagation();
      combobox.closeDropdown('keyboard');
      combobox.targetRef.current?.focus();
    }

    if (event.key === 'Tab' && !event.shiftKey && isOpen && onCreate) {
      event.preventDefault();
      createButtonRef.current?.focus();
    }
  };

  const renderOptions = () => (
    <Combobox.Options className="max-h-60 min-h-0 overflow-y-auto overscroll-contain">
      {options.map((option) => (
        <Combobox.Option
          key={option.value}
          value={option.value}
          active={selectedValues.includes(option.value)}
          aria-selected={selectedValues.includes(option.value)}
        >
          <span className="flex items-start justify-between gap-2">
            <span className="block min-w-0 wrap-anywhere whitespace-normal">{option.label}</span>
            {selectedValues.includes(option.value) && (
              <Check aria-hidden size={16} className="mt-0.5 shrink-0 text-accent" />
            )}
          </span>
        </Combobox.Option>
      ))}
    </Combobox.Options>
  );

  return (
    <div className={`flex min-w-0 items-center gap-1 ${className ?? ''}`}>
      <Combobox
        store={combobox}
        middlewares={{ shift: { padding: 8, crossAxis: true }, size: { padding: 8 } }}
        preventPositionChangeWhenVisible={false}
        onOptionSubmit={(value) => {
          onSelect(value);
          if (!isMultiple) {
            combobox.closeDropdown();
            combobox.targetRef.current?.focus();
          }
        }}
      >
        {isMultiple ? (
          <Combobox.DropdownTarget>
            <PillsInput
              size={size}
              className="min-w-0 flex-1"
              onClick={() => combobox.openDropdown()}
            >
              <Pill.Group>
                {selectedOptions.map((option) => (
                  <Pill
                    key={option.value}
                    classNames={WRAPPING_PILL_CLASS_NAMES}
                    withRemoveButton
                    removeButtonProps={{
                      // A percentage icon size makes the button's intrinsic width that of a
                      // default SVG, which stretches every pill to the full picker width.
                      iconSize: 14,
                      'aria-label': `${t('common.actions.remove')} ${option.label}`,
                      'aria-hidden': false,
                      tabIndex: 0,
                    }}
                    onRemove={() => onRemove?.(option.value)}
                  >
                    {option.label}
                  </Pill>
                ))}
                <Combobox.EventsTarget withExpandedAttribute>
                  <PillsInput.Field
                    ref={targetRef as Ref<HTMLInputElement>}
                    id={id}
                    aria-label={label}
                    // The field only receives keys, so beside pills it stays two pixels wide: it never
                    // wraps onto a line of its own, and its focus outline reads as a caret.
                    className={selectedOptions.length ? '!w-0.5 !min-w-0 !flex-none' : undefined}
                    placeholder={selectedOptions.length ? undefined : placeholder}
                    readOnly
                    onKeyDown={handleTargetKeyDown}
                  />
                </Combobox.EventsTarget>
              </Pill.Group>
            </PillsInput>
          </Combobox.DropdownTarget>
        ) : (
          <Combobox.Target targetType="button" withExpandedAttribute>
            <Button
              ref={targetRef as Ref<HTMLButtonElement>}
              id={id}
              aria-label={label}
              className="min-w-0 flex-1"
              size={size}
              onClick={() => combobox.toggleDropdown()}
              onKeyDown={handleTargetKeyDown}
              styles={{ inner: { justifyContent: 'space-between' }, label: { textAlign: 'start' } }}
              rightSection={
                isLoading ? (
                  <Loader size={12} role="status" />
                ) : (
                  <ChevronDown aria-hidden size={12} />
                )
              }
              aria-busy={isLoading}
            >
              {selectedOptions[0]?.label ?? <span className="text-muted">{placeholder}</span>}
            </Button>
          </Combobox.Target>
        )}
        <Combobox.Dropdown className="flex flex-col">
          {isSearchable && (
            <Combobox.Search
              className="shrink-0"
              aria-label={label}
              placeholder={searchPlaceholder}
              value={query}
              onChange={(event) => {
                setQuery(event.currentTarget.value);
                combobox.resetSelectedOption();
              }}
              onKeyDown={handleTargetKeyDown}
            />
          )}
          {renderOptions()}
          {onCreate && (
            <Combobox.Footer className="shrink-0">
              <Button
                ref={createButtonRef}
                variant="subtle"
                fullWidth
                size="xs"
                className="text-start"
                onMouseDown={(event) => event.preventDefault()}
                onClick={() => {
                  combobox.closeDropdown();
                  combobox.targetRef.current?.focus();
                  onCreate();
                }}
                onKeyDown={(event) => {
                  if (event.key === 'Escape' || (event.key === 'Tab' && event.shiftKey)) {
                    event.preventDefault();
                    event.stopPropagation();
                    if (event.key === 'Escape') combobox.closeDropdown();
                    combobox.targetRef.current?.focus();
                  }
                }}
              >
                {createLabel}
              </Button>
            </Combobox.Footer>
          )}
        </Combobox.Dropdown>
      </Combobox>
      {onClear && selectedValues.length > 0 && (
        <ActionIcon size="xs" aria-label={t('common.actions.clear')} onClick={onClear}>
          <X aria-hidden size={12} />
        </ActionIcon>
      )}
    </div>
  );
};
