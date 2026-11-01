import { InfoCircleOutlined } from '@ant-design/icons';
import { Tooltip } from 'antd';

interface Props {
  title: string;
}

export const ConnectionFieldTooltip = ({ title }: Props) => (
  <Tooltip className="cursor-pointer" title={title}>
    <span className="ml-2 text-muted" tabIndex={0}>
      <InfoCircleOutlined aria-label={title} />
    </span>
  </Tooltip>
);
