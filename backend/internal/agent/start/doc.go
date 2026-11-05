// Package start runs the verification agent as a single-instance background
// daemon: it spawns a detached process, holds an exclusive lock so a second
// agent cannot run from the same working directory, and supervises the
// capacity heartbeat and verification runner until a signal or a self-upgrade
// restart.
package start
