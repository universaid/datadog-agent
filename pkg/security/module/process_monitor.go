// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build linux
// +build linux

package module

import (
	sprobe "github.com/DataDog/datadog-agent/pkg/security/probe"
	"github.com/DataDog/datadog-agent/pkg/security/secl/model"
	"github.com/DataDog/datadog-agent/pkg/security/secl/rules"
	"time"
)

// ProcessMonitoring describes a process monitoring object
type ProcessMonitoring struct {
	module *Module
}

// ProcessEvent is an event sent by the ProcessMonitoring module
type ProcessEvent struct {
	*model.ProcessCacheEntry
	EventType string
	Date      time.Time
}

// HandleEvent implement the EventHandler interface
func (p *ProcessMonitoring) HandleEvent(event *sprobe.Event) {
	entry := event.ResolveProcessCacheEntry()
	if entry == nil {
		return
	}

	p.module.apiServer.SendProcessEvent(&ProcessEvent{
		ProcessCacheEntry: entry,
		EventType:         event.GetEventType().String(),
		Date:              event.Timestamp,
	})
}

// HandleCustomEvent implement the EventHandler interface
func (p *ProcessMonitoring) HandleCustomEvent(rule *rules.Rule, event *sprobe.CustomEvent) {
}

// NewProcessMonitoring creates a new ProcessMonitoring
func NewProcessMonitoring(module *Module) *ProcessMonitoring {
	return &ProcessMonitoring{
		module: module,
	}
}
