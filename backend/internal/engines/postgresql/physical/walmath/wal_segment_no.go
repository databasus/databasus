// Adapted from wal-g/internal/databases/postgres/wal_segment_no.go.
// Copyright 2017 Citus Data Inc. Licensed under the Apache License, Version 2.0.

package walmath

import "fmt"

type WalSegmentNo uint64

func NewWalSegmentNo(lsn LSN, segmentSizeBytes uint64) WalSegmentNo {
	return WalSegmentNo(uint64(lsn) / segmentSizeBytes)
}

func (walSegmentNo WalSegmentNo) Next() WalSegmentNo { return walSegmentNo + 1 }

func (walSegmentNo WalSegmentNo) FirstLSN(segmentSizeBytes uint64) LSN {
	return LSN(uint64(walSegmentNo) * segmentSizeBytes)
}

func (walSegmentNo WalSegmentNo) GetFilename(timeline uint32, segmentSizeBytes uint64) string {
	segmentsPerLogID := 0x100000000 / segmentSizeBytes
	return fmt.Sprintf(
		walFileFormat,
		timeline,
		uint64(walSegmentNo)/segmentsPerLogID,
		uint64(walSegmentNo)%segmentsPerLogID,
	)
}
