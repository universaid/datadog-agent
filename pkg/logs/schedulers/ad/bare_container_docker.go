// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build docker
// +build docker

package ad

import (
	"context"

	logsConfig "github.com/DataDog/datadog-agent/pkg/logs/config"
	"github.com/DataDog/datadog-agent/pkg/logs/internal/util"
	"github.com/DataDog/datadog-agent/pkg/util/containers"
	dockerutilpkg "github.com/DataDog/datadog-agent/pkg/util/docker"
	"github.com/DataDog/datadog-agent/pkg/util/log"
)

// bareContainerConfig constructs a LogsConfig for the given docker container,
// when LogWhat is LogContainers.  It also returns the appropriate name for the
// resulting source.
//
// This emulates the old behavior of the docker launcher in the definition of
// Source, Service, and the name of the LogSource.
func (s *Scheduler) bareContainerConfig(containerID string) (cfg *logsConfig.LogsConfig, sourceName string, err error) {
	ctx := context.TODO()
	var shortName string
	var standardService string

	// try to get shortName and standardService, but in the event of any errors just
	// log them and move on
	dockerutil, err := dockerutilpkg.GetDockerUtil()
	if err == nil {
		containerJSON, err := dockerutil.Inspect(ctx, containerID, false)
		if err == nil {
			imageName, err := dockerutil.ResolveImageName(ctx, containerJSON.Image)
			if err == nil {
				_, shortName, _, err = containers.SplitImageName(imageName)
				if err != nil {
					log.Debugf("Cannot parse image name %s: %s", imageName, err)
				}
			} else {
				log.Debugf("Could not resolve image name for %s: %s", containerID, err)
			}

			standardService = util.ServiceNameFromTags(
				containerJSON.Name,
				dockerutilpkg.ContainerIDToTaggerEntityName(containerID))
		} else {
			log.Warnf("Could not find container with id %s: %s", containerID, err)
		}
	} else {
		// dockerutil should always be available, as scheduler startup waits for it.
	}
	err = nil // all errors were logged above

	if shortName == "" {
		shortName = "docker"
	}
	if standardService == "" {
		standardService = shortName
	}

	cfg = &logsConfig.LogsConfig{
		Type:       logsConfig.DockerType,
		Source:     shortName,
		Service:    standardService,
		Identifier: containerID,
	}

	sourceName = cfg.Source
	return
}
