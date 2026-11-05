// Package sshtunnel forwards a local loopback port to a database that is only reachable from
// inside a customer's perimeter, through an SSH bastion.
//
// Callers hand the forwarder's local endpoint to whatever dials the database - a CLI tool's
// -h/-p arguments, a driver DSN - so nothing downstream needs to know a tunnel exists.
package sshtunnel
