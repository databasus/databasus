import { NumberInput } from '@mantine/core';
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
    <NumberInput
      id={id}
      min={minimum}
      value={isCleared ? '' : value}
      onChange={(entered) => {
        setIsCleared(typeof entered !== 'number');
        onChange(typeof entered === 'number' ? entered : minimum);
      }}
      onBlur={() => setIsCleared(false)}
      size="xs"
      className="w-[80px] max-w-[80px]"
      role="spinbutton"
      aria-valuenow={isCleared ? undefined : value}
      aria-valuemin={minimum}
    />
  );
};
