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

package controllers_test

import (
	"context"
	"fmt"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog/v2/klogr"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	configv1alpha1 "github.com/projectsveltos/cluster-api-feature-manager/api/v1alpha1"
	"github.com/projectsveltos/cluster-api-feature-manager/controllers"
)

const (
	serviceTemplate = `apiVersion: v1
kind: Service
metadata:
  name: service0
  namespace: %s
spec:
  selector:
    app.kubernetes.io/name: service0
  ports:
    - protocol: TCP
      port: 80
      targetPort: 9376
---
apiVersion: v1
kind: Service
metadata:
  name: service1
  namespace: %s
spec:
  selector:
    app.kubernetes.io/name: service1
  ports:
  - name: name-of-service-port
    protocol: TCP
    port: 80
    targetPort: http-web-svc
`

	deplTemplate = `apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx
  namespace: %s
spec:
  strategy:
    type: Recreate
  selector:
    matchLabels:
      app: nginx
  replicas: 3
  template:
    metadata:
      labels:
        app: nginx
    spec:
      containers:
      - name: nginx
        image: nginx
        ports:
        - containerPort: 80`
)

var _ = Describe("HandlersUtils", func() {
	var clusterSummary *configv1alpha1.ClusterSummary
	var clusterFeature *configv1alpha1.ClusterFeature
	var namespace string

	BeforeEach(func() {
		namespace = "reconcile" + randomString()

		cluster := &clusterv1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      upstreamClusterNamePrefix + randomString(),
				Namespace: namespace,
				Labels: map[string]string{
					"dc": "eng",
				},
			},
		}

		clusterFeature = &configv1alpha1.ClusterFeature{
			ObjectMeta: metav1.ObjectMeta{
				Name: clusterFeatureNamePrefix + randomString(),
			},
			Spec: configv1alpha1.ClusterFeatureSpec{
				ClusterSelector: selector,
			},
		}

		clusterSummaryName := controllers.GetClusterSummaryName(clusterFeature.Name, cluster.Name)
		clusterSummary = &configv1alpha1.ClusterSummary{
			ObjectMeta: metav1.ObjectMeta{
				Name:      clusterSummaryName,
				Namespace: cluster.Namespace,
			},
			Spec: configv1alpha1.ClusterSummarySpec{
				ClusterNamespace: cluster.Namespace,
				ClusterName:      cluster.Name,
			},
		}

		prepareForDeployment(clusterFeature, clusterSummary, cluster)

		// Get ClusterSummary so OwnerReference is set
		Expect(testEnv.Get(context.TODO(),
			types.NamespacedName{Namespace: clusterSummary.Namespace, Name: clusterSummary.Name}, clusterSummary)).To(Succeed())
	})

	AfterEach(func() {
		deleteResources(namespace, clusterFeature, clusterSummary)
	})

	It("addClusterSummaryLabel adds label with clusterSummary name", func() {
		role := &rbacv1.Role{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: namespace,
				Name:      randomString(),
			},
		}

		controllers.AddLabel(role, controllers.ClusterSummaryLabelName, clusterSummary.Name)
		Expect(role.Labels).ToNot(BeNil())
		Expect(len(role.Labels)).To(Equal(1))
		for k := range role.Labels {
			Expect(role.Labels[k]).To(Equal(clusterSummary.Name))
		}

		role.Labels = map[string]string{"reader": "ok"}
		controllers.AddLabel(role, controllers.ClusterSummaryLabelName, clusterSummary.Name)
		Expect(role.Labels).ToNot(BeNil())
		Expect(len(role.Labels)).To(Equal(2))
		found := false
		for k := range role.Labels {
			if role.Labels[k] == clusterSummary.Name {
				found = true
				break
			}
		}
		Expect(found).To(BeTrue())
	})

	It("createNamespace creates namespace", func() {
		initObjects := []client.Object{}

		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(initObjects...).Build()

		Expect(controllers.CreateNamespace(context.TODO(), c, clusterSummary, namespace)).To(BeNil())

		currentNs := &corev1.Namespace{}
		Expect(c.Get(context.TODO(), types.NamespacedName{Name: namespace}, currentNs)).To(Succeed())
	})

	It("createNamespace does not namespace in DryRun mode", func() {
		initObjects := []client.Object{}

		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(initObjects...).Build()

		clusterSummary.Spec.ClusterFeatureSpec.SyncMode = configv1alpha1.SyncModeDryRun
		Expect(controllers.CreateNamespace(context.TODO(), c, clusterSummary, namespace)).To(BeNil())

		currentNs := &corev1.Namespace{}
		err := c.Get(context.TODO(), types.NamespacedName{Name: namespace}, currentNs)
		Expect(err).ToNot(BeNil())
		Expect(apierrors.IsNotFound(err)).To(BeTrue())
	})

	It("createNamespace returns no error if namespace already exists", func() {
		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: namespace,
			},
		}
		initObjects := []client.Object{ns}

		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(initObjects...).Build()

		Expect(controllers.CreateNamespace(context.TODO(), c, clusterSummary, namespace)).To(BeNil())

		currentNs := &corev1.Namespace{}
		Expect(c.Get(context.TODO(), types.NamespacedName{Name: namespace}, currentNs)).To(Succeed())
	})

	It("deployContent in DryRun mode returns policies which will be created, updated and have conflicts", func() {
		services := fmt.Sprintf(serviceTemplate, namespace, namespace)
		depl := fmt.Sprintf(deplTemplate, namespace)

		clusterSummary.Spec.ClusterFeatureSpec.SyncMode = configv1alpha1.SyncModeDryRun

		secret := createSecretWithPolicy(namespace, randomString(), depl, services)
		Expect(testEnv.Client.Create(context.TODO(), secret)).To(Succeed())

		Expect(waitForObject(ctx, testEnv.Client, secret)).To(Succeed())
		Expect(addTypeInformationToObject(testEnv.Scheme(), clusterSummary)).To(Succeed())

		created, updated, conflict, err := controllers.DeployContent(context.TODO(), testEnv.Config, testEnv.Client, testEnv.Client,
			secret, map[string]string{"service": services}, clusterSummary, klogr.New())
		Expect(err).To(BeNil())
		Expect(len(created)).To(Equal(2))
		Expect(len(updated)).To(Equal(0))
		Expect(len(conflict)).To(Equal(0))

		// Create services
		elements := strings.Split(services, "---")
		for i := range elements {
			var policy *unstructured.Unstructured
			policy, err = controllers.GetUnstructured([]byte(elements[i]))
			Expect(err).To(BeNil())
			Expect(testEnv.Client.Create(context.TODO(), policy))
			Expect(waitForObject(ctx, testEnv.Client, policy)).To(Succeed())
		}

		created, updated, conflict, err = controllers.DeployContent(context.TODO(), testEnv.Config, testEnv.Client, testEnv.Client,
			secret, map[string]string{"service": services}, clusterSummary, klogr.New())
		Expect(err).To(BeNil())
		Expect(len(created)).To(Equal(0))
		Expect(len(updated)).To(Equal(2))
		Expect(len(conflict)).To(Equal(0))

		// Mark services as owned by a different secret
		elements = strings.Split(services, "---")
		for i := range elements {
			var policy *unstructured.Unstructured
			policy, err = controllers.GetUnstructured([]byte(elements[i]))
			Expect(err).To(BeNil())
			currentService := &corev1.Service{}
			Expect(testEnv.Client.Get(context.TODO(),
				types.NamespacedName{Namespace: policy.GetNamespace(), Name: policy.GetName()}, currentService)).To(Succeed())
			currentService.Labels = map[string]string{
				controllers.ReferenceLabelKind:      "Secret",
				controllers.ReferenceLabelName:      randomString(),
				controllers.ReferenceLabelNamespace: randomString(),
			}
			Expect(testEnv.Client.Update(context.TODO(), currentService)).To(Succeed())
		}

		// Wait for cache to be updated
		Eventually(func() bool {
			created, updated, conflict, err = controllers.DeployContent(context.TODO(), testEnv.Config, testEnv.Client, testEnv.Client,
				secret, map[string]string{"service": services}, clusterSummary, klogr.New())
			return err == nil &&
				len(created) == 0 && len(updated) == 0 && len(conflict) == 2
		}, timeout, pollingInterval).Should(BeTrue())
	})

	It("deployContentOfSecret deploys all policies contained in a ConfigMap", func() {
		services := fmt.Sprintf(serviceTemplate, namespace, namespace)
		depl := fmt.Sprintf(deplTemplate, namespace)

		secret := createSecretWithPolicy(namespace, randomString(), depl, services)

		Expect(testEnv.Client.Create(context.TODO(), secret)).To(Succeed())

		Expect(waitForObject(ctx, testEnv.Client, secret)).To(Succeed())

		Expect(addTypeInformationToObject(testEnv.Scheme(), clusterSummary)).To(Succeed())

		created, updated, _, err := controllers.DeployContentOfSecret(context.TODO(), testEnv.Config, testEnv.Client, testEnv.Client,
			secret, clusterSummary, klogr.New())
		Expect(err).To(BeNil())
		Expect(len(created) + len(updated)).To(Equal(3))
	})

	It("deployContentOfConfigMap deploys all policies contained in a Secret", func() {
		services := fmt.Sprintf(serviceTemplate, namespace, namespace)
		depl := fmt.Sprintf(deplTemplate, namespace)

		configMap := createConfigMapWithPolicy(namespace, randomString(), depl, services)

		Expect(testEnv.Client.Create(context.TODO(), configMap)).To(Succeed())

		Expect(waitForObject(ctx, testEnv.Client, configMap)).To(Succeed())

		Expect(addTypeInformationToObject(testEnv.Scheme(), clusterSummary)).To(Succeed())

		created, updated, _, err := controllers.DeployContentOfConfigMap(context.TODO(), testEnv.Config, testEnv.Client, testEnv.Client,
			configMap, clusterSummary, klogr.New())
		Expect(err).To(BeNil())
		Expect(len(created) + len(updated)).To(Equal(3))
	})

	It("undeployStaleResources does not remove resources in dryRun mode", func() {
		// Set ClusterSummary to be DryRun
		currentClusterSummary := &configv1alpha1.ClusterSummary{}
		Expect(testEnv.Get(context.TODO(),
			types.NamespacedName{Namespace: clusterSummary.Namespace, Name: clusterSummary.Name},
			currentClusterSummary)).To(Succeed())
		currentClusterSummary.Spec.ClusterFeatureSpec.SyncMode = configv1alpha1.SyncModeDryRun
		Expect(testEnv.Update(context.TODO(), currentClusterSummary)).To(Succeed())

		// Add list of GroupVersionKind this ClusterSummary has deployed in the CAPI Cluster
		// because of the PolicyRefs feature. This is used by UndeployStaleResources.
		err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			err := testEnv.Get(context.TODO(),
				types.NamespacedName{Namespace: clusterSummary.Namespace, Name: clusterSummary.Name},
				currentClusterSummary)
			if err != nil {
				return err
			}
			currentClusterSummary.Status.FeatureSummaries = []configv1alpha1.FeatureSummary{
				{
					FeatureID: configv1alpha1.FeatureResources,
					Status:    configv1alpha1.FeatureStatusProvisioned,
					DeployedGroupVersionKind: []string{
						"ClusterRole.v1.rbac.authorization.k8s.io",
					},
				},
			}
			return testEnv.Status().Update(context.TODO(), currentClusterSummary)
		})
		Expect(err).To(BeNil())

		configMapNs := randomString()
		viewClusterRoleName := randomString()
		configMap := createConfigMapWithPolicy(configMapNs, randomString(), fmt.Sprintf(viewClusterRole, viewClusterRoleName))

		// Create ClusterRole policy in the cluster, pretending it was created because of this ConfigMap and because
		// of this ClusterSummary (owner is ClusterFeature owning the ClusterSummary)
		clusterRole, err := controllers.GetUnstructured([]byte(fmt.Sprintf(viewClusterRole, viewClusterRoleName)))
		Expect(err).To(BeNil())
		clusterRole.SetLabels(map[string]string{
			controllers.ReferenceLabelKind:      string(configv1alpha1.ConfigMapReferencedResourceKind),
			controllers.ReferenceLabelName:      configMap.Name,
			controllers.ReferenceLabelNamespace: configMap.Namespace,
		})
		clusterRole.SetOwnerReferences([]metav1.OwnerReference{
			{Kind: configv1alpha1.ClusterFeatureKind, Name: clusterFeature.Name,
				UID: clusterFeature.UID, APIVersion: "config.projectsveltos.io/v1beta1"},
		})
		Expect(testEnv.Create(context.TODO(), clusterRole)).To(Succeed())
		Expect(waitForObject(ctx, testEnv.Client, clusterRole)).To(Succeed())

		deployedGKVs := controllers.GetDeployedGroupVersionKinds(currentClusterSummary, configv1alpha1.FeatureResources)
		Expect(deployedGKVs).ToNot(BeEmpty())

		// Because ClusterSummary is not referencing any ConfigMap/Resource and because test created a ClusterRole
		// pretending it was created by this ClusterSummary instance, UndeployStaleResources will remove no instance as
		// syncMode is dryRun and will report one instance (ClusterRole created above) would be undeployed
		undeploy, err := controllers.UndeployStaleResources(context.TODO(), testEnv.Config, testEnv.Client, testEnv.Client,
			currentClusterSummary, deployedGKVs, nil, klogr.New())
		Expect(err).To(BeNil())
		Expect(len(undeploy)).To(Equal(1))

		// Verify clusterRole is still present
		currentClusterRole := &rbacv1.ClusterRole{}
		Expect(testEnv.Get(context.TODO(), types.NamespacedName{Name: clusterRole.GetName()}, currentClusterRole)).To(BeNil())
	})

	It(`undeployStaleResources removes all policies created by ClusterSummary due to ConfigMaps not referenced anymore`, func() {
		configMapNs := randomString()
		viewClusterRoleName := randomString()
		configMap1 := createConfigMapWithPolicy(configMapNs, randomString(), fmt.Sprintf(viewClusterRole, viewClusterRoleName))
		editClusterRoleName := randomString()
		configMap2 := createConfigMapWithPolicy(configMapNs, randomString(), fmt.Sprintf(editClusterRole, editClusterRoleName))

		currentClusterSummary := &configv1alpha1.ClusterSummary{}
		Expect(testEnv.Get(context.TODO(),
			types.NamespacedName{Namespace: clusterSummary.Namespace, Name: clusterSummary.Name},
			currentClusterSummary)).To(Succeed())
		currentClusterSummary.Spec.ClusterFeatureSpec.PolicyRefs = []configv1alpha1.PolicyRef{
			{Namespace: configMapNs, Name: configMap1.Name, Kind: string(configv1alpha1.ConfigMapReferencedResourceKind)},
			{Namespace: configMapNs, Name: configMap2.Name, Kind: string(configv1alpha1.ConfigMapReferencedResourceKind)},
		}
		Expect(testEnv.Update(context.TODO(), currentClusterSummary)).To(Succeed())

		// Wait for cache to be updated
		Eventually(func() bool {
			err := testEnv.Get(context.TODO(),
				types.NamespacedName{Namespace: clusterSummary.Namespace, Name: clusterSummary.Name},
				currentClusterSummary)
			return err == nil &&
				currentClusterSummary.Spec.ClusterFeatureSpec.PolicyRefs != nil
		}, timeout, pollingInterval).Should(BeTrue())

		Expect(addTypeInformationToObject(testEnv.Scheme(), currentClusterSummary)).To(Succeed())

		clusterRoleName1 := controllers.GetPolicyName(viewClusterRoleName, currentClusterSummary)
		clusterRole1 := &rbacv1.ClusterRole{
			ObjectMeta: metav1.ObjectMeta{
				Name: clusterRoleName1,
				Labels: map[string]string{
					controllers.ReferenceLabelKind:      configMap1.Kind,
					controllers.ReferenceLabelNamespace: configMap1.Namespace,
					controllers.ReferenceLabelName:      configMap1.Name,
				},
			},
		}

		clusterRoleName2 := controllers.GetPolicyName(editClusterRoleName, currentClusterSummary)
		clusterRole2 := &rbacv1.ClusterRole{
			ObjectMeta: metav1.ObjectMeta{
				Name:      clusterRoleName2,
				Namespace: "default",
				Labels: map[string]string{
					controllers.ReferenceLabelKind:      configMap2.Kind,
					controllers.ReferenceLabelNamespace: configMap2.Namespace,
					controllers.ReferenceLabelName:      configMap2.Name,
				},
			},
		}

		// Add list of GroupVersionKind this ClusterSummary has deployed in the CAPI Cluster
		// because of the PolicyRefs feature. This is used by UndeployStaleResources.
		currentClusterSummary.Status.FeatureSummaries = []configv1alpha1.FeatureSummary{
			{
				FeatureID: configv1alpha1.FeatureResources,
				Status:    configv1alpha1.FeatureStatusProvisioned,
				DeployedGroupVersionKind: []string{
					"ClusterRole.v1.rbac.authorization.k8s.io",
				},
			},
		}

		Expect(testEnv.Client.Status().Update(context.TODO(), currentClusterSummary)).To(Succeed())
		Expect(testEnv.Client.Create(context.TODO(), clusterRole1)).To(Succeed())
		Expect(testEnv.Client.Create(context.TODO(), clusterRole2)).To(Succeed())
		Expect(waitForObject(ctx, testEnv.Client, clusterRole2)).To(Succeed())

		currentClusterFeature := &configv1alpha1.ClusterFeature{}
		Expect(testEnv.Get(context.TODO(),
			types.NamespacedName{Name: clusterFeature.Name},
			currentClusterFeature)).To(Succeed())

		addOwnerReference(context.TODO(), testEnv.Client, clusterRole1, currentClusterFeature)
		addOwnerReference(context.TODO(), testEnv.Client, clusterRole2, currentClusterFeature)

		Expect(addTypeInformationToObject(testEnv.Scheme(), clusterRole1)).To(Succeed())
		Expect(addTypeInformationToObject(testEnv.Scheme(), clusterRole2)).To(Succeed())

		currentClusterRoles := map[string]configv1alpha1.Resource{}
		clusterRoleResource1 := &configv1alpha1.Resource{
			Name:  clusterRole1.Name,
			Kind:  clusterRole1.GroupVersionKind().Kind,
			Group: clusterRole1.GetObjectKind().GroupVersionKind().Group,
		}
		currentClusterRoles[controllers.GetPolicyInfo(clusterRoleResource1)] = *clusterRoleResource1
		clusterRoleResource2 := &configv1alpha1.Resource{
			Name:  clusterRole2.Name,
			Kind:  clusterRole2.GroupVersionKind().Kind,
			Group: clusterRole2.GetObjectKind().GroupVersionKind().Group,
		}
		currentClusterRoles[controllers.GetPolicyInfo(clusterRoleResource2)] = *clusterRoleResource2

		deployedGKVs := controllers.GetDeployedGroupVersionKinds(currentClusterSummary, configv1alpha1.FeatureResources)
		Expect(deployedGKVs).ToNot(BeEmpty())
		// undeployStaleResources finds all instances of policies deployed because of clusterSummary and
		// removes the stale ones.
		_, err := controllers.UndeployStaleResources(context.TODO(), testEnv.Config, testEnv.Client, testEnv.Client,
			currentClusterSummary, deployedGKVs, currentClusterRoles, klogr.New())
		Expect(err).To(BeNil())

		// Consistently loop so testEnv Cache is synced
		Consistently(func() error {
			// Since ClusterSummary is referencing configMap, expect ClusterRole to not be deleted
			currentClusterRole := &rbacv1.ClusterRole{}
			return testEnv.Get(context.TODO(),
				types.NamespacedName{Name: clusterRoleName1}, currentClusterRole)
		}, timeout, pollingInterval).Should(BeNil())

		// Consistently loop so testEnv Cache is synced
		Consistently(func() error {
			// Since ClusterSummary is referencing configMap, expect Policy to not be deleted
			currentClusterRole := &rbacv1.ClusterRole{}
			return testEnv.Get(context.TODO(),
				types.NamespacedName{Name: clusterRoleName2}, currentClusterRole)
		}, timeout, pollingInterval).Should(BeNil())

		currentClusterSummary.Spec.ClusterFeatureSpec.PolicyRefs = nil
		delete(currentClusterRoles, controllers.GetPolicyInfo(clusterRoleResource1))
		delete(currentClusterRoles, controllers.GetPolicyInfo(clusterRoleResource2))

		_, err = controllers.UndeployStaleResources(context.TODO(), testEnv.Config, testEnv.Client, testEnv.Client,
			currentClusterSummary, deployedGKVs, currentClusterRoles, klogr.New())
		Expect(err).To(BeNil())

		// Eventual loop so testEnv Cache is synced
		Eventually(func() bool {
			// Since ClusterSummary is not referencing configMap with ClusterRole, expect ClusterRole to be deleted
			currentClusterRole := &rbacv1.ClusterRole{}
			err = testEnv.Get(context.TODO(),
				types.NamespacedName{Name: clusterRoleName1}, currentClusterRole)
			return err != nil && apierrors.IsNotFound(err)
		}, timeout, pollingInterval).Should(BeTrue())

		// Eventual loop so testEnv Cache is synced
		Eventually(func() bool {
			// Since ClusterSummary is not referencing configMap with ClusterRole, expect ClusterRole to be deleted
			currentClusterRole := &rbacv1.ClusterRole{}
			err = testEnv.Get(context.TODO(),
				types.NamespacedName{Name: clusterRoleName2}, currentClusterRole)
			return err != nil && apierrors.IsNotFound(err)
		}, timeout, pollingInterval).Should(BeTrue())
	})
})
