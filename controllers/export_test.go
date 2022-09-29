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

package controllers

var (
	GetMatchingClusters          = (*ClusterFeatureReconciler).getMatchingClusters
	UpdateClusterSummaries       = (*ClusterFeatureReconciler).updateClusterSummaries
	CreateClusterSummary         = (*ClusterFeatureReconciler).createClusterSummary
	UpdateClusterSummary         = (*ClusterFeatureReconciler).updateClusterSummary
	DeleteClusterSummary         = (*ClusterFeatureReconciler).deleteClusterSummary
	GetMachinesForCluster        = (*ClusterFeatureReconciler).getMachinesForCluster
	IsClusterReadyToBeConfigured = (*ClusterFeatureReconciler).isClusterReadyToBeConfigured
	UpdateClusterConfiguration   = (*ClusterFeatureReconciler).updateClusterConfiguration
	CleanClusterConfiguration    = (*ClusterFeatureReconciler).cleanClusterConfiguration
	CleanClusterReports          = (*ClusterFeatureReconciler).cleanClusterReports
	UpdateClusterReports         = (*ClusterFeatureReconciler).updateClusterReports
	UpdateClusterSummarySyncMode = (*ClusterFeatureReconciler).updateClusterSummarySyncMode

	RequeueClusterFeatureForCluster = (*ClusterFeatureReconciler).requeueClusterFeatureForCluster
	RequeueClusterFeatureForMachine = (*ClusterFeatureReconciler).requeueClusterFeatureForMachine
)

var (
	IsFeatureDeployed              = (*ClusterSummaryReconciler).isFeatureDeployed
	GetHash                        = (*ClusterSummaryReconciler).getHash
	UpdateFeatureStatus            = (*ClusterSummaryReconciler).updateFeatureStatus
	DeployFeature                  = (*ClusterSummaryReconciler).deployFeature
	UndeployFeature                = (*ClusterSummaryReconciler).undeployFeature
	UpdateDeployedGroupVersionKind = (*ClusterSummaryReconciler).updateDeployedGroupVersionKind
	GetCurrentReferences           = (*ClusterSummaryReconciler).getCurrentReferences
	IsPaused                       = (*ClusterSummaryReconciler).isPaused
	ShouldReconcile                = (*ClusterSummaryReconciler).shouldReconcile
	UpdateChartMap                 = (*ClusterSummaryReconciler).updateChartMap
	ShouldRedeploy                 = (*ClusterSummaryReconciler).shouldRedeploy
	CanRemoveFinalizer             = (*ClusterSummaryReconciler).canRemoveFinalizer

	ConvertResultStatus               = (*ClusterSummaryReconciler).convertResultStatus
	RequeueClusterSummaryForReference = (*ClusterSummaryReconciler).requeueClusterSummaryForReference
	RequeueClusterSummaryForCluster   = (*ClusterSummaryReconciler).requeueClusterSummaryForCluster
)

var (
	CreatFeatureHandlerMaps = creatFeatureHandlerMaps
	GetHandlersForFeature   = getHandlersForFeature
	GenericDeploy           = genericDeploy
	GenericUndeploy         = genericUndeploy

	GetClusterSummary            = getClusterSummary
	GetSecretData                = getSecretData
	GetKubernetesClient          = getKubernetesClient
	AddLabel                     = addLabel
	CreateNamespace              = createNamespace
	GetEntryKey                  = getEntryKey
	DeployContentOfConfigMap     = deployContentOfConfigMap
	DeployContentOfSecret        = deployContentOfSecret
	DeployContent                = deployContent
	GetPolicyName                = getPolicyName
	GetPolicyInfo                = getPolicyInfo
	UndeployStaleResources       = undeployStaleResources
	GetDeployedGroupVersionKinds = getDeployedGroupVersionKinds

	ResourcesHash   = resourcesHash
	GetResourceRefs = getResourceRefs

	HelmHash                                 = helmHash
	ShouldInstall                            = shouldInstall
	ShouldUninstall                          = shouldUninstall
	ShouldUpgrade                            = shouldUpgrade
	UpdateChartsInClusterConfiguration       = updateChartsInClusterConfiguration
	UpdateStatusForReferencedHelmReleases    = updateStatusForReferencedHelmReleases
	UpdateStatusForNonReferencedHelmReleases = updateStatusForNonReferencedHelmReleases
	CreateReportForUnmanagedHelmRelease      = createReportForUnmanagedHelmRelease
	UpdateClusterReportWithHelmReports       = updateClusterReportWithHelmReports
	HandleCharts                             = handleCharts
)

type (
	ReleaseInfo = releaseInfo
)

var (
	GetClusterFeatureOwner = getClusterFeatureOwner
	GetUnstructured        = getUnstructured
	AddOwnerReference      = addOwnerReference
	RemoveOwnerReference   = removeOwnerReference
)

var (
	IsTemplate          = isTemplate
	PropValue           = propValue
	GetObject           = getObject
	InstantiateTemplate = instantiateTemplate
)

var (
	Insert = (*Set).insert
	Erase  = (*Set).erase
	Len    = (*Set).len
	Items  = (*Set).items
	Has    = (*Set).has
)

var (
	GetClusterReportName = getClusterReportName
	CreateKubeconfig     = createKubeconfig
)
