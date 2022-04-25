// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build !docker
// +build !docker

package ad

import (
	"errors"

	logsConfig "github.com/DataDog/datadog-agent/pkg/logs/config"
)

// bareContainerConfig constructs a LogsConfig for the given docker container,
// when LogWhat is LogContainers.  It also returns the appropriate name for the
// resulting source.
//
// This emulates the old behavior of the docker launcher in the definition of
// Source, Service, and the name of the LogSource.
func (s *Scheduler) bareContainerConfig(containerID string) (cfg *logsConfig.LogsConfig, sourceName string, err error) {
	err = errors.New("docker not enabled")
	return
}
