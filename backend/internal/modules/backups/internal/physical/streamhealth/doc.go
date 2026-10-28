// Package streamhealth decides how healthy WAL archiving of a streaming database
// is: the verdict with its reasons and notices, the keeping-up summary and the
// streamer state. It also owns every threshold those decisions use, including
// the ones the WAL streamer's notifications and the WAL stream manager act on,
// so the screen and the notifications agree about when a condition began.
package streamhealth
