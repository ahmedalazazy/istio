// Copyright Istio Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package xds

import (
	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"

	"istio.io/istio/pilot/pkg/model"
	"istio.io/istio/pilot/pkg/networking/util"
	"istio.io/istio/pkg/config"
	"istio.io/istio/pkg/config/schema/gvk"
)

type RdsGenerator struct {
	Server *DiscoveryServer
}

var _ model.XdsResourceGenerator = &RdsGenerator{}

// Map of all configs that do not impact RDS
var skippedRdsConfigs = map[config.GroupVersionKind]struct{}{
	gvk.WorkloadEntry:         {},
	gvk.WorkloadGroup:         {},
	gvk.AuthorizationPolicy:   {},
	gvk.RequestAuthentication: {},
	gvk.PeerAuthentication:    {},
	gvk.Secret:                {},
}

func rdsNeedsPush(req *model.PushRequest) bool {
	if req == nil {
		return true
	}
	if !req.Full {
		// RDS only handles full push
		return false
	}
	// If none set, we will always push
	if len(req.ConfigsUpdated) == 0 {
		return true
	}
	for config := range req.ConfigsUpdated {
		if _, f := skippedRdsConfigs[config.Kind]; !f {
			return true
		}
	}
	return false
}

func (c RdsGenerator) Generate(proxy *model.Proxy, push *model.PushContext, w *model.WatchedResource,
	req *model.PushRequest) (model.Resources, model.XdsLogDetails, error) {
	if !rdsNeedsPush(req) {
		return nil, model.DefaultXdsLogDetails, nil
	}
	rawRoutes := c.Server.ConfigGenerator.BuildHTTPRoutes(proxy, push, w.ResourceNames)
	resources := model.Resources{}
	for _, c := range rawRoutes {
		resources = append(resources, &discovery.Resource{
			Name:     c.Name,
			Resource: util.MessageToAny(c),
		})
	}
	return resources, model.DefaultXdsLogDetails, nil
}

func (c *RdsGenerator) GenerateDeltas(proxy *model.Proxy, push *model.PushContext, updates *model.PushRequest,
	w *model.WatchedResource) (model.Resources, []string, model.XdsLogDetails, bool, error) {
	res, logs, err := c.Generate(proxy, push, w, updates)
	return res, nil, logs, false, err
}
