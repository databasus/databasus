import { ActionIcon, Popover, TextInput } from '@mantine/core';
import { InlineDateTimePicker } from '@mantine/dates';
import dayjs, { type Dayjs } from 'dayjs';
import customParseFormat from 'dayjs/plugin/customParseFormat';
import { CalendarDays, X } from 'lucide-react';
import { useEffect, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { getUserTimeFormat } from '@/shared/time';

dayjs.extend(customParseFormat);

interface Props {
  id: string;
  value?: Dayjs;
  onChange: (value?: Dayjs) => void;
}

export const PhysicalRestoreTargetTimeField = ({ id, value, onChange }: Props) => {
  const { t } = useTranslation();
  const inputRef = useRef<HTMLInputElement>(null);
  const [isOpen, setIsOpen] = useState(false);
  const [draft, setDraft] = useState('');
  const [calendarValue, setCalendarValue] = useState<string | null>(null);
  const displayFormat = getUserTimeFormat().format.replace(':mm', ':mm:ss');
  const formattedValue = value?.format(displayFormat) ?? '';

  useEffect(() => {
    setDraft(formattedValue);
  }, [formattedValue]);

  const openCalendar = () => {
    setCalendarValue(value?.format('YYYY-MM-DD HH:mm:ss') ?? null);
    setIsOpen(true);
  };

  const commitDraft = () => {
    if (!draft.trim()) {
      onChange(undefined);
      return;
    }

    const parsed = dayjs(draft, [displayFormat, 'YYYY-MM-DD HH:mm:ss', 'YYYY-MM-DD HH:mm'], true);
    if (parsed.isValid()) onChange(parsed);
    else setDraft(formattedValue);
  };

  return (
    <Popover
      withRoles={false}
      closeOnEscape={false}
      opened={isOpen}
      onChange={setIsOpen}
      position="bottom-start"
      withinPortal={false}
      trapFocus
      returnFocus
    >
      <Popover.Target>
        <TextInput
          ref={inputRef}
          id={id}
          aria-haspopup="dialog"
          aria-expanded={isOpen}
          aria-controls={isOpen ? `${id}-calendar` : undefined}
          value={draft}
          onChange={(event) => setDraft(event.currentTarget.value)}
          onClick={openCalendar}
          onKeyDown={(event) => {
            if (event.key === 'Escape' && isOpen) {
              event.stopPropagation();
              setIsOpen(false);
              setDraft(formattedValue);
            }

            if (event.key === 'ArrowDown') {
              event.preventDefault();
              openCalendar();
            }

            if (event.key === 'Enter') {
              event.preventDefault();
              commitDraft();
              setIsOpen(false);
            }
          }}
          onBlur={commitDraft}
          placeholder={t('backups.physical.restore.latestAvailable')}
          className="w-full max-w-[260px] grow"
          rightSection={
            value ? (
              <ActionIcon
                size="xs"
                aria-label={t('common.actions.clear')}
                onMouseDown={(event) => event.preventDefault()}
                onClick={() => {
                  onChange(undefined);
                  setDraft('');
                  setCalendarValue(null);
                }}
              >
                <X aria-hidden size={14} />
              </ActionIcon>
            ) : (
              <CalendarDays aria-hidden size={14} />
            )
          }
        />
      </Popover.Target>
      <Popover.Dropdown
        role="dialog"
        id={`${id}-calendar`}
        aria-label={t('backups.physical.restore.targetTime')}
        onKeyDown={(event) => {
          if (event.key === 'Escape') {
            event.stopPropagation();
            setIsOpen(false);
            inputRef.current?.focus();
          }
        }}
      >
        <InlineDateTimePicker
          fullWidth={false}
          value={calendarValue}
          onChange={setCalendarValue}
          withSeconds
          submitButtonProps={{ 'aria-label': t('common.actions.ok') }}
          onSubmit={() => {
            onChange(calendarValue ? dayjs(calendarValue) : undefined);
            setIsOpen(false);
            inputRef.current?.focus();
          }}
        />
      </Popover.Dropdown>
    </Popover>
  );
};
