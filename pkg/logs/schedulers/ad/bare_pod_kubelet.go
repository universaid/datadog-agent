// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build kubelet
// +build kubelet

package ad

import (
	"context"
	"fmt"

	logsConfig "github.com/DataDog/datadog-agent/pkg/logs/config"
	"github.com/DataDog/datadog-agent/pkg/logs/internal/util"
	"github.com/DataDog/datadog-agent/pkg/util/containers"
	"github.com/DataDog/datadog-agent/pkg/util/kubernetes/kubelet"
	"github.com/DataDog/datadog-agent/pkg/util/log"
)

// barePodConfig constructs a LogsConfig for the given docker container,
// when LogWhat is LogPods.  It also returns the appropriate name for the
// resulting source.
//
// This emulates the old behavior of the kubernetes launcher in the definition of
// Source, Service, and the name of the LogSource.
func (s *Scheduler) barePodConfig(containerID, serviceID string) (cfg *logsConfig.LogsConfig, sourceName string, err error) {
	ctx := context.TODO()
	var shortName string
	var standardService string

	// try to get shortName and standardService, but in the event of any errors just
	// log them and move on
	kubeutil, err := kubelet.GetKubeUtil()
	if err == nil {
		pod, err := kubeutil.GetPodForContainerID(ctx, serviceID)
		if err == nil {
			container, err := kubeutil.GetStatusForContainerID(pod, serviceID)
			if err == nil {
				containerSpec, err := kubeutil.GetSpecForContainerName(pod, container.Name)
				if err == nil {
					_, shortName, _, err = containers.SplitImageName(containerSpec.Image)
					if err != nil {
						log.Debugf("Cannot parse image name: %v", err)
					}
				} else {
					log.Warnf("Could not get container spec for %s: %v", serviceID, err)
				}

				sourceName = fmt.Sprintf("%s/%s/%s",
					pod.Metadata.Namespace,
					pod.Metadata.Name,
					container.Name)

				entityID, err := kubelet.KubeContainerIDToTaggerEntityID(container.ID)
				if err != nil {
					standardService = util.ServiceNameFromTags(container.Name, entityID)
				} else {
					log.Debugf("Could not get Tagger entityID for container %s: %s", container.ID, err)
				}
			} else {
				log.Warnf("Could not get status for container %s: %s", serviceID, err)
			}
		} else {
			log.Warnf("Could not get pod for container %s: %s", serviceID, err)
		}
	} else {
		// kubeutil should always be available, as scheduler startup waits for it.
	}

	if shortName == "" {
		shortName = "kubernetes"
	}
	if standardService == "" {
		standardService = shortName
	}
	if sourceName == "" {
		sourceName = shortName
	}

	cfg = &logsConfig.LogsConfig{
		Type:       logsConfig.DockerType,
		Source:     shortName,
		Service:    standardService,
		Identifier: containerID,
	}

	return
}
