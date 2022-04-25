// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build !kubelet
// +build !kubelet

package ad

import (
	"errors"

	logsConfig "github.com/DataDog/datadog-agent/pkg/logs/config"
)

// barePodConfig constructs a LogsConfig for the given docker container,
// when LogWhat is LogPods.  It also returns the appropriate name for the
// resulting source.
//
// This emulates the old behavior of the kubernetes launcher in the definition of
// Source, Service, and the name of the LogSource.
func (s *Scheduler) barePodConfig(containerID, serviceID string) (cfg *logsConfig.LogsConfig, sourceName string, err error) {
	err = errors.New("kubelet not enabled")
	return
}
