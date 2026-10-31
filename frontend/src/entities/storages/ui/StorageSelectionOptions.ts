export interface StorageSelectionOptions {
  workspaceId: string;
  selectedStorageId?: string;
  canClear: boolean;
  onSelect: (storageId?: string) => void;
}
