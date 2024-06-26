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
	"bytes"
	"context"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	configv1alpha1 "github.com/projectsveltos/addon-controller/api/v1alpha1"
	"github.com/projectsveltos/libsveltos/lib/utils"
)

// The TemplateResource namespace can be specified or it will inherit the cluster namespace
func getTemplateResourceNamespace(clusterSummary *configv1alpha1.ClusterSummary,
	ref *configv1alpha1.TemplateResourceRef) string {

	namespace := ref.Resource.Namespace
	if namespace == "" {
		// Use cluster namespace
		namespace = clusterSummary.Spec.ClusterNamespace
	}

	return namespace
}

// Resources referenced in the management cluster can have their name expressed in function
// of cluster information (clusterNamespace, clusterName, clusterType)
func getTemplateResourceName(clusterSummary *configv1alpha1.ClusterSummary,
	ref *configv1alpha1.TemplateResourceRef) (string, error) {

	// Accept name that are templates
	templateName := getTemplateName(clusterSummary.Spec.ClusterNamespace, clusterSummary.Spec.ClusterName,
		string(clusterSummary.Spec.ClusterType))
	tmpl, err := template.New(templateName).Option("missingkey=error").Funcs(sprig.FuncMap()).Parse(ref.Resource.Name)
	if err != nil {
		return "", err
	}

	var buffer bytes.Buffer

	if err := tmpl.Execute(&buffer,
		struct{ ClusterNamespace, ClusterName string }{
			ClusterNamespace: clusterSummary.Spec.ClusterNamespace,
			ClusterName:      clusterSummary.Spec.ClusterName}); err != nil {
		return "", errors.Wrapf(err, "error executing template")
	}
	return buffer.String(), nil
}

// collectTemplateResourceRefs collects clusterSummary.Spec.ClusterProfileSpec.TemplateResourceRefs
// from management cluster
func collectTemplateResourceRefs(ctx context.Context, clusterSummary *configv1alpha1.ClusterSummary,
) (map[string]*unstructured.Unstructured, error) {

	if clusterSummary.Spec.ClusterProfileSpec.TemplateResourceRefs == nil {
		return nil, nil
	}

	restConfig := getManagementClusterConfig()

	result := make(map[string]*unstructured.Unstructured)
	for i := range clusterSummary.Spec.ClusterProfileSpec.TemplateResourceRefs {
		ref := clusterSummary.Spec.ClusterProfileSpec.TemplateResourceRefs[i]
		ref.Resource.Namespace = getTemplateResourceNamespace(clusterSummary, &ref)
		var err error
		ref.Resource.Name, err = getTemplateResourceName(clusterSummary, &ref)
		if err != nil {
			return nil, err
		}

		dr, err := utils.GetDynamicResourceInterface(restConfig, ref.Resource.GroupVersionKind(), ref.Resource.Namespace)
		if err != nil {
			return nil, err
		}

		var u *unstructured.Unstructured
		u, err = dr.Get(ctx, ref.Resource.Name, metav1.GetOptions{})
		if err != nil {
			if apierrors.IsNotFound(err) {
				continue
			}
			return nil, err
		}

		result[ref.Identifier] = u
	}

	return result, nil
}
