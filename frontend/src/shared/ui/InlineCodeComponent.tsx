import type { ReactNode } from 'react';

interface Props {
  // Optional because Trans injects the children when the component is passed in `components`.
  children?: ReactNode;
}

// A monospace font alone barely sets code apart from prose, so inline code also gets a background.
export const InlineCodeComponent = ({ children }: Props) => (
  <code className="rounded bg-code box-decoration-clone px-1 py-0.5 text-[0.9em]">{children}</code>
);
