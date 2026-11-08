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
    <button
      type="button"
      className="inline-flex max-w-full cursor-pointer items-center gap-2 text-left text-sm text-accent"
      aria-expanded={isShowAdvanced}
      aria-controls={regionId}
      onClick={onToggle}
    >
      <span>{label}</span>

      {isShowAdvanced ? (
        <ChevronUp aria-hidden size={12} className="shrink-0" />
      ) : (
        <ChevronDown aria-hidden size={12} className="shrink-0" />
      )}
    </button>
  </div>
);
