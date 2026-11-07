import { ActionIcon, Button, ComboboxPopover, type MantineSize } from '@mantine/core';
import { ChevronDown, X } from 'lucide-react';
import type { ComponentPropsWithoutRef, ReactNode, Ref } from 'react';
import { useTranslation } from 'react-i18next';

interface Props<Value extends string | number> extends Omit<
  ComponentPropsWithoutRef<'button'>,
  'value' | 'defaultValue' | 'onChange' | 'size' | 'color'
> {
  data: {
    value: Value;
    label: string;
    disabled?: boolean;
  }[];
  value?: Value;
  onChange: (value: Value) => void;
  renderOption?: (value: Value) => ReactNode;
  onClear?: () => void;
  placeholder?: string;
  size?: MantineSize;
  isLoading?: boolean;
  hasError?: boolean;
  ref?: Ref<HTMLButtonElement>;
}

export const WrappingSelect = <Value extends string | number>({
  data,
  value,
  onChange,
  renderOption,
  onClear,
  placeholder,
  size = 'xs',
  isLoading = false,
  hasError = false,
  className,
  style,
  disabled,
  ...buttonProps
}: Props<Value>) => {
  const { t } = useTranslation();
  const selectedOption = data.find((option) => option.value === value);

  return (
    <div className={`flex min-w-0 items-center gap-1 ${className ?? ''}`} style={style}>
      <ComboboxPopover
        data={data}
        value={value ?? null}
        allowDeselect={false}
        renderOption={
          renderOption
            ? ({ option }) => {
                const item = data.find((candidate) => candidate.value === option.value);
                return item ? renderOption(item.value) : option.label;
              }
            : undefined
        }
        onChange={(selected) => {
          if (selected !== null) onChange(selected);
        }}
        comboboxProps={{ width: 'target', disabled: disabled || isLoading }}
      >
        <ComboboxPopover.Target>
          <Button
            {...buttonProps}
            type="button"
            size={size}
            fullWidth
            disabled={disabled}
            loading={isLoading}
            aria-invalid={hasError || undefined}
            styles={{
              root: hasError ? { borderColor: 'var(--app-danger)' } : undefined,
              inner: { justifyContent: 'space-between' },
              label: { minWidth: 0, textAlign: 'start' },
            }}
            rightSection={
              <ChevronDown aria-hidden size={12} className="inline-block shrink-0 align-middle" />
            }
            aria-busy={isLoading}
          >
            {selectedOption ? (
              (renderOption?.(selectedOption.value) ?? selectedOption.label)
            ) : (
              <span className="text-muted">{placeholder}</span>
            )}
          </Button>
        </ComboboxPopover.Target>
      </ComboboxPopover>
      {onClear && value !== undefined && (
        <ActionIcon
          aria-label={t('common.actions.clear')}
          disabled={disabled || isLoading}
          onClick={onClear}
        >
          <X aria-hidden size={12} className="inline-block shrink-0 align-middle" />
        </ActionIcon>
      )}
    </div>
  );
};
