import { Button } from '@mantine/core';
import { ChevronDown, ChevronUp } from 'lucide-react';

interface Props {
  label: string;
  isShowAdvanced: boolean;
  onToggle: () => void;
  regionId?: string;
}

export const AdvancedSettingsToggleComponent = ({
  label,
  isShowAdvanced,
  onToggle,
  regionId,
}: Props) => (
  <div className="mt-4 mb-1 flex items-center">
    <Button
      variant="transparent"
      size="xs"
      className="!h-auto !px-0 text-left !whitespace-normal"
      aria-expanded={isShowAdvanced}
      aria-controls={regionId}
      onClick={onToggle}
    >
      <span className="mr-2">{label}</span>

      {isShowAdvanced ? (
        <ChevronUp aria-hidden size={12} className="inline-block shrink-0 align-middle" />
      ) : (
        <ChevronDown aria-hidden size={12} className="inline-block shrink-0 align-middle" />
      )}
    </Button>
  </div>
);
