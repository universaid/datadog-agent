// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package configresolver

import (
	"context"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/DataDog/datadog-agent/pkg/autodiscovery/integration"
	"github.com/DataDog/datadog-agent/pkg/autodiscovery/listeners"
	"github.com/DataDog/datadog-agent/pkg/autodiscovery/providers/names"
	"github.com/DataDog/datadog-agent/pkg/util/containers"
	"github.com/DataDog/datadog-agent/pkg/util/log"

	yaml "gopkg.in/yaml.v2"
)

type variableGetter func(ctx context.Context, key string, svc listeners.Service) (string, error)

var templateVariables = map[string]variableGetter{
	"host":     getHost,
	"pid":      getPid,
	"port":     getPort,
	"hostname": getHostname,
	"env":      getEnvvar,
	"extra":    getAdditionalTplVariables,
	"kube":     getAdditionalTplVariables,
}

type fakeService struct{}

func (f *fakeService) GetServiceID() string {
	return ""
}

func (f *fakeService) GetTaggerEntity() string {
	return ""
}

func (f *fakeService) GetADIdentifiers(context.Context) ([]string, error) {
	return nil, fmt.Errorf("fakeService")
}

func (f *fakeService) GetHosts(context.Context) (map[string]string, error) {
	return nil, fmt.Errorf("fakeService")
}

func (f *fakeService) GetPorts(context.Context) ([]listeners.ContainerPort, error) {
	return nil, fmt.Errorf("fakeService")
}

func (f *fakeService) GetTags() ([]string, error) {
	return nil, fmt.Errorf("fakeService")
}

func (f *fakeService) GetPid(context.Context) (int, error) {
	return 0, fmt.Errorf("fakeService")
}

func (f *fakeService) GetHostname(context.Context) (string, error) {
	return "", fmt.Errorf("fakeService")
}

func (f *fakeService) IsReady(context.Context) bool {
	return false
}

func (f *fakeService) GetCheckNames(context.Context) []string {
	return nil
}

func (f *fakeService) HasFilter(containers.FilterType) bool {
	return false
}

func (f *fakeService) GetExtraConfig(string) (string, error) {
	return "", fmt.Errorf("fakeService")
}

// SubstituteTemplateEnvVars replaces %%ENV_VARIABLE%% from environment
// variables in the config init, instances, and logs config.
// When there is an error, it continues replacing. When there are multiple
// errors, the one returned is the one that happened first.
func SubstituteTemplateEnvVars(config *integration.Config) error {
	var fakeSvc fakeService
	return substituteTemplateVariables(context.Background(), config, &fakeSvc, nil)
}

// Resolve takes a template and a service and generates a config with
// valid connection info and relevant tags.
func Resolve(tpl integration.Config, svc listeners.Service) (integration.Config, error) {
	ctx := context.TODO()
	// Copy original template
	resolvedConfig := integration.Config{
		Name:            tpl.Name,
		Instances:       make([]integration.Data, len(tpl.Instances)),
		InitConfig:      make(integration.Data, len(tpl.InitConfig)),
		MetricConfig:    tpl.MetricConfig,
		LogsConfig:      tpl.LogsConfig,
		ADIdentifiers:   tpl.ADIdentifiers,
		ClusterCheck:    tpl.ClusterCheck,
		Provider:        tpl.Provider,
		ServiceID:       svc.GetServiceID(),
		NodeName:        tpl.NodeName,
		Source:          tpl.Source,
		MetricsExcluded: svc.HasFilter(containers.MetricsFilter),
		LogsExcluded:    svc.HasFilter(containers.LogsFilter),
	}
	copy(resolvedConfig.InitConfig, tpl.InitConfig)
	copy(resolvedConfig.Instances, tpl.Instances)

	// Ignore the config from file if it's overridden by an empty config
	// or by a different config for the same check
	if tpl.Provider == names.File && svc.GetCheckNames(ctx) != nil {
		checkNames := svc.GetCheckNames(ctx)
		lenCheckNames := len(checkNames)
		if lenCheckNames == 0 || (lenCheckNames == 1 && checkNames[0] == "") {
			// Empty check names on k8s annotations or container labels override the check config from file
			// Used to deactivate unneeded OOTB autodiscovery checks defined in files
			// The checkNames slice is considered empty also if it contains one single empty string
			return resolvedConfig, fmt.Errorf("ignoring config from %s: another empty config is defined with the same AD identifier: %v", tpl.Source, tpl.ADIdentifiers)
		}
		for _, checkName := range checkNames {
			if tpl.Name == checkName {
				// Ignore config from file when the same check is activated on the same service via other config providers (k8s annotations or container labels)
				return resolvedConfig, fmt.Errorf("ignoring config from %s: another config is defined for the check %s", tpl.Source, tpl.Name)
			}
		}

	}

	if resolvedConfig.IsCheckConfig() && !svc.IsReady(ctx) {
		return resolvedConfig, errors.New("unable to resolve, service not ready")
	}

	tags, err := svc.GetTags()
	if err != nil {
		return resolvedConfig, fmt.Errorf("couldn't get tags for service '%s', err: %w", svc.GetServiceID(), err)
	}

	var postProcessor func(interface{}) error

	if !tpl.IgnoreAutodiscoveryTags {
		postProcessor = tagsAdder(tags)
	}

	if err := substituteTemplateVariables(ctx, &resolvedConfig, svc, postProcessor); err != nil {
		return resolvedConfig, err
	}

	return resolvedConfig, nil
}

// substituteTemplateVariables replaces %%VARIABLES%% in the config init,
// instances, and logs config.
// When there is an error, it stops processing.
func substituteTemplateVariables(ctx context.Context, config *integration.Config, svc listeners.Service, postProcessor func(interface{}) error) error {
	var err error

	for _, toResolve := range listDataToResolve(config) {
		var pp func(interface{}) error
		if toResolve.dtype == dataInstance {
			pp = postProcessor
		}
		*toResolve.data, err = resolveDataWithTemplateVars(ctx, *toResolve.data, svc, pp)
		if err != nil {
			return err
		}
	}

	return nil
}

type dataType int

const (
	dataInit dataType = iota
	dataInstance
	dataLogs
	dataMetric
)

type dataToResolve struct {
	data  *integration.Data
	dtype dataType
}

func listDataToResolve(config *integration.Config) []dataToResolve {
	res := []dataToResolve{
		{
			data:  &config.InitConfig,
			dtype: dataInit,
		},
	}

	for i := 0; i < len(config.Instances); i++ {
		res = append(res, dataToResolve{
			data:  &config.Instances[i],
			dtype: dataInstance,
		})
	}

	if config.IsLogConfig() {
		res = append(res, dataToResolve{
			data:  &config.LogsConfig,
			dtype: dataLogs,
		})
	}

	return res
}

func resolveDataWithTemplateVars(ctx context.Context, data integration.Data, svc listeners.Service, postProcessor func(interface{}) error) ([]byte, error) {
	if len(data) == 0 {
		return nil, nil
	}

	var tree interface{}

	// Percent character is not allowed in unquoted yaml strings.
	data2 := strings.ReplaceAll(string(data), "%%", "‰")
	if err := yaml.Unmarshal([]byte(data2), &tree); err != nil {
		return data, err
	}

	type treePointer struct {
		get func() interface{}
		set func(interface{})
	}

	stack := []treePointer{
		{
			get: func() interface{} {
				return tree
			},
			set: func(x interface{}) {
				tree = x
			},
		},
	}

	for len(stack) > 0 {
		n := len(stack) - 1
		top := stack[n]
		stack = stack[:n]

		switch elem := top.get().(type) {

		case map[interface{}]interface{}:
			for k, v := range elem {
				k2, v2 := k, v
				stack = append(stack, treePointer{
					get: func() interface{} {
						return v2
					},
					set: func(x interface{}) {
						elem[k2] = x
					},
				})
			}

		case []interface{}:
			for i, v := range elem {
				i2, v2 := i, v
				stack = append(stack, treePointer{
					get: func() interface{} {
						return v2
					},
					set: func(x interface{}) {
						elem[i2] = x
					},
				})
			}

		case string:
			s, err := resolveStringWithTemplateVars(ctx, elem, svc)
			if err != nil {
				return data, err
			}
			top.set(s)

		case int, bool:

		default:
			return data, fmt.Errorf("Unknown type: %T", elem)
		}
	}

	if postProcessor != nil {
		if err := postProcessor(tree); err != nil {
			return data, err
		}
	}

	return yaml.Marshal(&tree)
}

var varPattern = regexp.MustCompile(`‰(.+?)(?:_(.+?))?‰`)

func resolveStringWithTemplateVars(ctx context.Context, in string, svc listeners.Service) (out interface{}, err error) {
	varIndexes := varPattern.FindAllStringSubmatchIndex(in, -1)

	if len(varIndexes) == 0 {
		return in, nil
	}

	var sb strings.Builder

	sb.WriteString(in[:varIndexes[0][0]])
	for i := range varIndexes {
		if i != 0 {
			sb.WriteString(in[varIndexes[i-1][1]:varIndexes[i][0]])
		}

		varName := in[varIndexes[i][2]:varIndexes[i][3]]
		varKey := ""
		if varIndexes[i][4] != -1 {
			varKey = in[varIndexes[i][4]:varIndexes[i][5]]
		}

		if f, found := templateVariables[varName]; found {
			resolvedVar, e := f(ctx, varKey, svc)
			if e != nil {
				err = e
			}
			sb.WriteString(resolvedVar)
		} else {
			endTagIdx := varIndexes[i][5]
			if endTagIdx == -1 {
				endTagIdx = varIndexes[i][3]
			}
			return out, fmt.Errorf("unable to add tags for service '%s', err: invalid %%%%%s%%%% tag", svc.GetServiceID(), in[varIndexes[i][2]:endTagIdx])
		}
	}
	sb.WriteString(in[varIndexes[len(varIndexes)-1][1]:])

	out = sb.String()

	if len(varIndexes) == 1 &&
		varIndexes[0][0] == 0 &&
		varIndexes[0][1] == len(in) {

		if i, e := strconv.ParseInt(out.(string), 0, 64); e == nil {
			return i, err
		}
		if b, e := strconv.ParseBool(out.(string)); e == nil {
			return b, err
		}
	}

	return
}

func tagsAdder(tags []string) func(interface{}) error {
	return func(tree interface{}) error {
		if typedTree, ok := tree.(map[interface{}]interface{}); ok {
			tagList, _ := typedTree["tags"].([]string)
			// Use a set to remove duplicates
			tagSet := make(map[string]struct{})
			for _, t := range tagList {
				tagSet[t] = struct{}{}
			}
			for _, t := range tags {
				tagSet[t] = struct{}{}
			}
			typedTree["tags"] = make([]string, len(tagSet))
			i := 0
			for k := range tagSet {
				typedTree["tags"].([]string)[i] = k
				i++
			}
		}
		return nil
	}
}

func getHost(ctx context.Context, tplVar string, svc listeners.Service) (string, error) {
	hosts, err := svc.GetHosts(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to extract IP address for container %s, ignoring it. Source error: %s", svc.GetServiceID(), err)
	}
	if len(hosts) == 0 {
		return "", fmt.Errorf("no network found for container %s, ignoring it", svc.GetServiceID())
	}

	// a network was specified
	if ip, ok := hosts[tplVar]; ok {
		return ip, nil
	}
	log.Debugf("Network %q not found, trying bridge IP instead", tplVar)

	// otherwise use fallback policy
	ip, err := getFallbackHost(hosts)
	if err != nil {
		return "", fmt.Errorf("failed to resolve IP address for container %s, ignoring it. Source error: %s", svc.GetServiceID(), err)
	}

	return ip, nil
}

// getFallbackHost implements the fallback strategy to get a service's IP address
// the current strategy is:
// 		- if there's only one network we use its IP
// 		- otherwise we look for the bridge net and return its IP address
// 		- if we can't find it we fail because we shouldn't try and guess the IP address
func getFallbackHost(hosts map[string]string) (string, error) {
	if len(hosts) == 1 {
		for _, host := range hosts {
			return host, nil
		}
	}

	bridgeIP, bridgeIsPresent := hosts["bridge"]
	if bridgeIsPresent {
		return bridgeIP, nil
	}

	return "", errors.New("not able to determine which network is reachable")
}

// getPort returns ports of the service
func getPort(ctx context.Context, tplVar string, svc listeners.Service) (string, error) {
	ports, err := svc.GetPorts(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to extract port list for container %s, ignoring it. Source error: %s", svc.GetServiceID(), err)
	} else if len(ports) == 0 {
		return "", fmt.Errorf("no port found for container %s - ignoring it", svc.GetServiceID())
	}

	if len(tplVar) == 0 {
		return strconv.Itoa(ports[len(ports)-1].Port), nil
	}

	idx, err := strconv.Atoi(tplVar)
	if err != nil {
		// The template variable is not an index so try to lookup port by name.
		for _, port := range ports {
			if port.Name == tplVar {
				return strconv.Itoa(port.Port), nil
			}
		}
		return "", fmt.Errorf("port %s not found, skipping container %s", tplVar, svc.GetServiceID())
	}
	if len(ports) <= idx {
		return "", fmt.Errorf("index given for the port template var is too big, skipping container %s", svc.GetServiceID())
	}
	return strconv.Itoa(ports[idx].Port), nil
}

// getPid returns the process identifier of the service
func getPid(ctx context.Context, _ string, svc listeners.Service) (string, error) {
	pid, err := svc.GetPid(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get pid for service %s, skipping config - %s", svc.GetServiceID(), err)
	}
	return strconv.Itoa(pid), nil
}

// getHostname returns the hostname of the service, to be used
// when the IP is unavailable or erroneous
func getHostname(ctx context.Context, _ string, svc listeners.Service) (string, error) {
	name, err := svc.GetHostname(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get hostname for service %s, skipping config - %s", svc.GetServiceID(), err)
	}
	return name, nil
}

// getAdditionalTplVariables returns listener-specific template variables.
// It resolves template variables prefixed with kube_ or extra_
// Even though it gets the data from the same listener method GetExtraConfig, the kube_ and extra_
// prefixes are customer facing, we support both of them for a better user experience depending on
// the AD listener and what the template variable represents.
func getAdditionalTplVariables(_ context.Context, tplVar string, svc listeners.Service) (string, error) {
	value, err := svc.GetExtraConfig(tplVar)
	if err != nil {
		return "", fmt.Errorf("failed to get extra info for service %s, skipping config - %s", svc.GetServiceID(), err)
	}
	return value, nil
}

// getEnvvar returns a system environment variable if found
func getEnvvar(_ context.Context, envVar string, svc listeners.Service) (string, error) {
	if len(envVar) == 0 {
		return "", fmt.Errorf("envvar name is missing, skipping service %s", svc.GetServiceID())
	}
	value, found := os.LookupEnv(envVar)
	if !found {
		return "", fmt.Errorf("failed to retrieve envvar %s, skipping service %s", envVar, svc.GetServiceID())
	}
	return value, nil
}
