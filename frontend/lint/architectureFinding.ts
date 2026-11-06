export interface ArchitectureFinding {
  rule: string;
  source: string;
  target?: string;
  line?: number;
  column?: number;
}
