// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package fileorsocket

// LogFrom is the answer this package provides.
type LogFrom int

const (
	// File indicates the container should be tailed by file
	File = iota

	// Socket indicates the container should be tailed by socket
	Socket
)

// Decider carries the information necessary to decide whether to log from file
// or socket.
type Decider struct {
	// if TailFromFile is true, tail from file unless a registry entry suggests
	// otherwise file
	TailFromFile bool

	// if ForceTailingFromFile is true, always tail from file, even if registry
	// has a socket offset
	ForceTailFromFile bool

	// SocketInRegistry indicates whether there is a registry entry for this source
	// using a socket.
	SocketInRegistry bool
}

// Decide determines whether to log from file or socket.
//
// The current logic is as follows:
//
// - Socket is the default, so if TailFromFile is not set => Socket
// - Forcing tailing from file ignores the registry => File
// - If a registry entry exists for the socket => Socket
// - File
func (d Decider) Decide() LogFrom {
	if !d.TailFromFile {
		return Socket
	}

	if d.ForceTailFromFile {
		return File
	}

	if d.SocketInRegistry {
		return Socket
	}

	return File
}
