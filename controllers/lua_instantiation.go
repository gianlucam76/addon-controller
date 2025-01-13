/*
Copyright 2024. projectsveltos.io. All rights reserved.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"encoding/json"
	"fmt"

	luastrings "github.com/chai2010/glua-strings"
	"github.com/go-logr/logr"
	luajson "github.com/layeh/gopher-json"
	lua "github.com/yuin/gopher-lua"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"

	libsveltosv1beta1 "github.com/projectsveltos/libsveltos/api/v1beta1"
	logs "github.com/projectsveltos/libsveltos/lib/logsettings"
)

type luaResult struct {
	// Resources is a list of Kubernetes resources
	Resources string `json:"resources"`
}

func getLuaHelperFunctions() string {
	return `
local json = require("json")
local strings = require("strings")

function getResource(resources, key)
  if resources ~= nil and resources[key] then
    return resources[key]
  else
    return nil
  end	
end

function getLabel(resource, key)
  if resource ~= nil and resource.metadata.labels ~= nil and resource.metadata.labels[key] then
    return resource.metadata.labels[key]
  else
    return ""
  end	
end


function getAnnotation(resource, key)
  if resource ~= nil and resource.metadata.annotations ~= nil and resource.metadata.annotations[key] then
    return resource.metadata.annotations[key]
  else
    return ""
  end	
end

function toJSON(resource)
  local jsonStr = json.encode(resource)
  print(jsonStr)
end

`
}

func instantiateWithLuaScript(ctx context.Context, config *rest.Config, c client.Client,
	clusterType libsveltosv1beta1.ClusterType, clusterNamespace, clusterName, script string,
	mgmtResources map[string]*unstructured.Unstructured, logger logr.Logger) (string, error) {

	if script == "" {
		return "", nil
	}

	luaCode := getLuaHelperFunctions()

	if luaConfigMap := getLuaConfigMap(); luaConfigMap != "" {
		configMap, err := collectLuaConfigMap(ctx)
		if err != nil {
			return "", err
		}

		for k := range configMap.Data {
			luaCode += configMap.Data[k]
			luaCode += "\n"
		}
	}

	luaCode += script

	objects, err := fecthClusterObjects(ctx, config, c, clusterNamespace, clusterName, clusterType, logger)
	if err != nil {
		return "", err
	}

	if mgmtResources != nil {
		objects.MgmtResources = make(map[string]map[string]interface{})
		for k := range mgmtResources {
			logger.V(logs.LogDebug).Info(fmt.Sprintf("using mgmt resource %s %s/%s with identifier %s",
				mgmtResources[k].GetKind(), mgmtResources[k].GetNamespace(), mgmtResources[k].GetName(), k))
			objects.MgmtResources[k] = mgmtResources[k].UnstructuredContent()
		}
	}

	// Create a new Lua state
	l := lua.NewState()
	defer l.Close()

	l.PreloadModule("json", luajson.Loader)
	l.PreloadModule("strings", luastrings.Loader)

	// Load the Lua code
	if err := l.DoString(luaCode); err != nil {
		logger.V(logs.LogInfo).Info(fmt.Sprintf("doString failed: %v", err))
		return "", err
	}

	argTable := l.NewTable()
	for key, resource := range objects.MgmtResources {
		lValue := mapToTable(resource)
		argTable.RawSetString(key, lValue)
	}

	l.SetGlobal("resources", argTable)

	if err := l.CallByParam(lua.P{
		Fn:      l.GetGlobal("evaluate"), // name of Lua function
		NRet:    1,                       // number of returned values
		Protect: true,                    // return err or panic
	}, argTable); err != nil {
		logger.V(logs.LogInfo).Info(fmt.Sprintf("failed to call evaluate function: %s", err.Error()))
		return "", err
	}

	lv := l.Get(-1)
	tbl, ok := lv.(*lua.LTable)
	if !ok {
		logger.V(logs.LogInfo).Info(luaTableError)
		return "", fmt.Errorf("%s", luaTableError)
	}

	goResult := toGoValue(tbl)
	resultJson, err := json.Marshal(goResult)
	if err != nil {
		logger.V(logs.LogInfo).Info(fmt.Sprintf("failed to marshal result: %v", err))
		return "", err
	}

	var result luaResult
	err = json.Unmarshal(resultJson, &result)
	if err != nil {
		logger.V(logs.LogInfo).Info(fmt.Sprintf("failed to marshal result: %v", err))
		return "", err
	}

	return result.Resources, nil
}
