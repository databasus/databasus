import { DownOutlined, UpOutlined } from '@ant-design/icons';
import { Button } from 'antd';

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
      type="link"
      size="small"
      className="!h-auto !px-0 text-left !whitespace-normal"
      aria-expanded={isShowAdvanced}
      aria-controls={regionId}
      onClick={onToggle}
    >
      <span className="mr-2">{label}</span>

      {isShowAdvanced ? (
        <UpOutlined aria-hidden style={{ fontSize: '12px' }} />
      ) : (
        <DownOutlined aria-hidden style={{ fontSize: '12px' }} />
      )}
    </Button>
  </div>
);
