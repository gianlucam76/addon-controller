//go:build ignore
// +build ignore

/*
Copyright 2022. projectsveltos.io. All rights reserved.

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

package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

const (
	prometheusTemplate = `// Generated by *go generate* - DO NOT EDIT
/*
Copyright 2022. projectsveltos.io. All rights reserved.

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

package prometheus

// Prometheus Operator namespace name
const Namespace = {{ printf "\"%s\"" .Namespace }}

// Prometheus operator deployment name
const Deployment = {{ printf "\"%s\"" .Deployment }}

var {{ .ExportedName }}YAML = []byte({{- printf "%s" .YAML -}})
`
)

const namespaceName = "monitoring"

func generatePrometheus(filename, outputFilename, exportedName string) {
	fileAbs, err := filepath.Abs(filename)
	if err != nil {
		panic(err)
	}

	content, err := ioutil.ReadFile(fileAbs)
	if err != nil {
		panic(err)
	}
	contentStr := string(content)
	contentStr = strings.ReplaceAll(contentStr, "`", "")

	contentStr = changeNamespace(contentStr)
	deploymentName := getDeploymentName(contentStr)

	contentStr = "`" + contentStr + "`"

	// Find the output.
	crd, err := os.Create(outputFilename + ".go")
	if err != nil {
		panic(err)
	}
	defer crd.Close()

	// Store file contents.
	type Info struct {
		YAML         string
		Namespace    string
		Deployment   string
		ExportedName string
	}
	mi := Info{
		YAML:         contentStr,
		Deployment:   deploymentName,
		ExportedName: exportedName,
		Namespace:    namespaceName,
	}

	// Generate template.
	manifest := template.Must(template.New("prometheus-generate").Parse(prometheusTemplate))
	if err := manifest.Execute(crd, mi); err != nil {
		panic(err)
	}
}

func main() {
	prometheusConfigurationFile := "../prometheus/prometheus.yaml"

	generatePrometheus(prometheusConfigurationFile, "prometheus", "Prometheus")
}

func changeNamespace(content string) string {
	namespace := "namespace: default"

	index := strings.Index(content, namespace)
	if index == -1 {
		panic(fmt.Errorf("did not find namespace: default"))
	}

	newNamespace := fmt.Sprintf("namespace: %s", namespaceName)
	content = strings.ReplaceAll(content, namespace, newNamespace)
	return content
}

func getDeploymentName(content string) string {
	ns := getKindSection(content, "Deployment")
	name := "name:"

	index := strings.Index(ns, name)
	if index == -1 {
		panic(fmt.Errorf("did not find Deployment name"))
	}

	var deploymentName string
	_, err := fmt.Sscanf(ns[index+len(name):], "%s", &deploymentName)
	if err != nil {
		panic(err)
	}
	return deploymentName
}

func getKindSection(content, kind string) string {
	section := "kind: " + kind
	newContent := ""
	s := bufio.NewScanner(strings.NewReader(content))
	copy := false
	for s.Scan() {
		read_line := s.Text()
		if strings.Contains(read_line, section) {
			copy = true
		}

		if copy {
			newContent += read_line + "\n"
			if read_line == "---" {
				return newContent
			}
		}
	}

	return newContent
}