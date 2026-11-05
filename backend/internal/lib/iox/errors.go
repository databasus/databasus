package iox

import "errors"

var (
	ErrReadRetryBudgetExhausted = errors.New("read retries exhausted")

	ErrReaderClosed = errors.New("reader is closed")

	// ErrRangeNotHonoured means a resumed read was answered with a different byte span than it
	// asked for, which some backends do by replying with the whole object. Replaying bytes that
	// were already handed out corrupts the stream for a stateful decryptor, so it is terminal.
	ErrRangeNotHonoured = errors.New("range request not honoured")

	// ErrStreamTruncated means the source ended before the size it declared. Retrying is still
	// worthwhile: the usual cause is a severed connection, not a shorter file.
	ErrStreamTruncated = errors.New("stream ended before the declared size")
)
