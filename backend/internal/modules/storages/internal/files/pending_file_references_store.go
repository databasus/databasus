package files

import "context"

func (s *Store) GetPendingFileReferences(ctx context.Context, prefix string) ([]StoredFileReference, error) {
	return s.repository.ListReferencesByPrefix(ctx, prefix)
}
