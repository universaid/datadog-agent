// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build linux
// +build linux

package events

import "time"

// ProcessEvent represents a process event collected by system-probe
type ProcessEvent struct {
	*Process
	EventType string    `json:"EventType"`
	Date      time.Time `json:"date"`
}

// FileEvent holds information about the binary executed by a process
type FileEvent struct {
	PathName string `json:"PathnameStr"`
}

// ArgsEntry holds information about a process command-line arguments
type ArgsEntry struct {
	Values []string `json:"Values"`
}

// Process holds metadata about a process
type Process struct {
	PID       int32     `json:"Pid"`
	PPID      int32     `json:"PPid"`
	UID       int32     `json:"UID"`
	GID       int32     `json:"GID"`
	User      string    `json:"User"`
	Group     string    `json:"Group"`
	FileEvent FileEvent `json:"FileEvent"`
	ArgsEntry ArgsEntry `json:"ArgsEntry"`
	ForkTime  time.Time `json:"ForkTime"`
	ExecTime  time.Time `json:"ExecTime"`
	ExitTime  time.Time `json:"ExitTime"`
}
