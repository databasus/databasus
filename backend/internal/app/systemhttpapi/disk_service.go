package systemhttpapi

import (
	"context"
	"fmt"
)

func (s *Service) GetDiskUsage(context.Context) (GetDiskUsageResponseObject, error) {
	usage, err := s.inspectDisk(s.dataDirectory)
	if err != nil {
		return nil, fmt.Errorf("inspect data volume: %w", err)
	}
	return GetDiskUsage200JSONResponse{
		TotalSpaceBytes: usage.TotalBytes,
		UsedSpaceBytes:  usage.UsedBytes,
		FreeSpaceBytes:  usage.FreeBytes,
	}, nil
}
