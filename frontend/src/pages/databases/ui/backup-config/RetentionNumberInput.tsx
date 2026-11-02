import { InputNumber } from 'antd';
import { useState } from 'react';

interface Props {
  id: string;
  minimum: number;
  value: number;
  onChange: (value: number) => void;
}

export const RetentionNumberInput = ({ id, minimum, value, onChange }: Props) => {
  const [isCleared, setIsCleared] = useState(false);

  return (
    <InputNumber
      id={id}
      min={minimum}
      value={isCleared ? null : value}
      onChange={(entered) => {
        setIsCleared(entered === null);
        onChange(entered ?? minimum);
      }}
      onBlur={() => setIsCleared(false)}
      size="small"
      className="w-[80px] max-w-[80px]"
    />
  );
};
