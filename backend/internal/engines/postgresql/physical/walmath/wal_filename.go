// Adapted from wal-g/internal/databases/postgres/timeline.go.
// Copyright 2017 Citus Data Inc. Licensed under the Apache License, Version 2.0.

package walmath

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

func ParseWALFilenameTimeline(name string) (uint32, error) {
	timelineID, _, err := parseWalSegmentFields(name)

	return timelineID, err
}

func ParseWALFilename(name string, segmentSize uint64) (timelineID uint32, logSegNo uint64, err error) {
	timelineID, logSegmentFields, err := parseWalSegmentFields(name)
	if err != nil {
		return 0, 0, err
	}

	if segmentSize == 0 || 0x100000000%segmentSize != 0 {
		return 0, 0, IncorrectLogSegNoError{Filename: name}
	}

	logSegNoHi := logSegmentFields >> hexUint32Bits
	logSegNoLo := logSegmentFields & math.MaxUint32

	segmentsPerLogID := 0x100000000 / segmentSize
	if logSegNoLo >= segmentsPerLogID {
		return 0, 0, IncorrectLogSegNoError{Filename: name}
	}

	logSegNo = logSegNoHi*segmentsPerLogID + logSegNoLo

	return timelineID, logSegNo, nil
}

func parseWalSegmentFields(name string) (timelineID uint32, logSegmentFields uint64, err error) {
	if len(name) != walSegmentFilenameLength {
		return 0, 0, NotWalFilenameError{Filename: name}
	}

	parsedTimelineID, err := strconv.ParseUint(name[0:8], 0x10, hexUint32Bits)
	if err != nil {
		return 0, 0, NotWalFilenameError{Filename: name}
	}

	logSegmentFields, err = strconv.ParseUint(name[8:walSegmentFilenameLength], 0x10, hexUint64Bits)
	if err != nil {
		return 0, 0, NotWalFilenameError{Filename: name}
	}

	return uint32(parsedTimelineID), logSegmentFields, nil
}

func IsWalFilename(filename string, segmentSize uint64) bool {
	_, _, err := ParseWALFilename(filename, segmentSize)
	return err == nil
}

func FormatHistoryFilename(timeline uint32) string {
	return fmt.Sprintf(walHistoryFileFormat, timeline)
}

func ParseHistoryFilename(name string) (int, error) {
	trimmed := strings.TrimSuffix(name, ".history")
	if trimmed == name {
		return 0, fmt.Errorf("not a history filename: %q", name)
	}
	timelineID, err := strconv.ParseUint(trimmed, 16, 32)
	if err != nil {
		return 0, fmt.Errorf("parse timeline from %q: %w", name, err)
	}
	return int(timelineID), nil
}
