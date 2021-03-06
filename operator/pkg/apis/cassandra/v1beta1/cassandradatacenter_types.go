// Copyright DataStax, Inc.
// Please see the included license file for details.

package v1beta1

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/Jeffail/gabs"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/datastax/cass-operator/operator/pkg/serverconfig"
	"github.com/datastax/cass-operator/operator/pkg/utils"
)

const (
	// ClusterLabel is the operator's label for the cluster name
	ClusterLabel = "cassandra.datastax.com/cluster"

	// DatacenterLabel is the operator's label for the datacenter name
	DatacenterLabel = "cassandra.datastax.com/datacenter"

	// SeedNodeLabel is the operator's label for the seed node state
	SeedNodeLabel = "cassandra.datastax.com/seed-node"

	// RackLabel is the operator's label for the rack name
	RackLabel = "cassandra.datastax.com/rack"

	// RackLabel is the operator's label for the rack name
	CassOperatorProgressLabel = "cassandra.datastax.com/operator-progress"

	// CassNodeState
	CassNodeState = "cassandra.datastax.com/node-state"

	// Progress states for status
	ProgressUpdating ProgressState = "Updating"
	ProgressReady    ProgressState = "Ready"
)

// This type exists so there's no chance of pushing random strings to our progress status
type ProgressState string

const (
	defaultConfigBuilderImage     = "datastax/cass-config-builder:1.0.1"
	ubi_defaultConfigBuilderImage = "datastax/cass-config-builder:1.0.1-ubi7"

	cassandra_3_11_6     = "datastax/cassandra-mgmtapi-3_11_6:v0.1.5"
	cassandra_4_0_0      = "datastax/cassandra-mgmtapi-4_0_0:v0.1.5"
	ubi_cassandra_3_11_6 = "datastax/cassandra:3.11.6-ubi7"
	ubi_cassandra_4_0_0  = "datastax/cassandra:4.0-ubi7"

	dse_6_8_0     = "datastax/dse-server:6.8.0"
	dse_6_8_1     = "datastax/dse-server:6.8.1"
	ubi_dse_6_8_0 = "datastax/dse-server:6.8.0-ubi7"
	ubi_dse_6_8_1 = "datastax/dse-server:6.8.1-ubi7"

	EnvBaseImageOs = "BASE_IMAGE_OS"
)

// getImageForServerVersion tries to look up a known image for a server type and version number.
// In the event that no image is found, an error is returned
func getImageForServerVersion(server, version string) (string, error) {
	baseImageOs := os.Getenv(EnvBaseImageOs)

	var imageCalc func(string) (string, bool)
	var img string
	var success bool
	var errMsg string

	if baseImageOs == "" {
		imageCalc = getImageForDefaultBaseOs
		errMsg = fmt.Sprintf("server '%s' and version '%s' do not work together", server, version)
	} else {
		// if this operator was compiled using a UBI base image
		// such as registry.access.redhat.com/ubi7/ubi-minimal:7.8
		// then we use specific cassandra and init container coordinates
		// that are built accordingly
		errMsg = fmt.Sprintf("server '%s' and version '%s', along with the specified base OS '%s', do not work together", server, version, baseImageOs)
		imageCalc = getImageForUniversalBaseOs
	}

	img, success = imageCalc(server + "-" + version)
	if !success {
		return "", fmt.Errorf(errMsg)
	}

	return img, nil
}

func getImageForDefaultBaseOs(sv string) (string, bool) {
	switch sv {
	case "dse-6.8.0":
		return dse_6_8_0, true
	case "dse-6.8.1":
		return dse_6_8_1, true
	case "cassandra-3.11.6":
		return cassandra_3_11_6, true
	case "cassandra-4.0.0":
		return cassandra_4_0_0, true
	}
	return "", false
}

func getImageForUniversalBaseOs(sv string) (string, bool) {
	switch sv {
	case "dse-6.8.0":
		return ubi_dse_6_8_0, true
	case "dse-6.8.1":
		return ubi_dse_6_8_1, true
	case "cassandra-3.11.6":
		return ubi_cassandra_3_11_6, true
	case "cassandra-4.0.0":
		return ubi_cassandra_4_0_0, true
	}
	return "", false
}

type CassandraUser struct {
	SecretName string `json:"secretName"`
	Superuser  bool   `json:"superuser"`
}

// CassandraDatacenterSpec defines the desired state of a CassandraDatacenter
// +k8s:openapi-gen=true
type CassandraDatacenterSpec struct {
	// Important: Run "mage operator:sdkGenerate" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags:
	// https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html

	// Desired number of Cassandra server nodes
	// +kubebuilder:validation:Minimum=1
	Size int32 `json:"size"`

	// Version string for config builder,
	// used to generate Cassandra server configuration
	// +kubebuilder:validation:Enum="6.8.0";"6.8.1";"3.11.6";"4.0.0"
	ServerVersion string `json:"serverVersion"`

	// Cassandra server image name.
	// More info: https://kubernetes.io/docs/concepts/containers/images
	ServerImage string `json:"serverImage,omitempty"`

	// Server type: "cassandra" or "dse"
	// +kubebuilder:validation:Enum=cassandra;dse
	ServerType string `json:"serverType"`

	// Config for the server, in YAML format
	Config json.RawMessage `json:"config,omitempty"`

	// Config for the Management API certificates
	ManagementApiAuth ManagementApiAuthConfig `json:"managementApiAuth,omitempty"`

	// Kubernetes resource requests and limits, per pod
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`

	// A list of the named racks in the datacenter, representing independent failure domains. The
	// number of racks should match the replication factor in the keyspaces you plan to create, and
	// the number of racks cannot easily be changed once a datacenter is deployed.
	Racks []Rack `json:"racks,omitempty"`

	// Describes the persistent storage request of each server node
	StorageConfig StorageConfig `json:"storageConfig"`

	// A list of pod names that need to be replaced.
	ReplaceNodes []string `json:"replaceNodes,omitempty"`

	// The name by which CQL clients and instances will know the cluster. If the same
	// cluster name is shared by multiple Datacenters in the same Kubernetes namespace,
	// they will join together in a multi-datacenter cluster.
	// +kubebuilder:validation:MinLength=2
	ClusterName string `json:"clusterName"`

	// A stopped CassandraDatacenter will have no running server pods, like using "stop" with
	// traditional System V init scripts. Other Kubernetes resources will be left intact, and volumes
	// will re-attach when the CassandraDatacenter workload is resumed.
	Stopped bool `json:"stopped,omitempty"`

	// Container image for the config builder init container.
	ConfigBuilderImage string `json:"configBuilderImage,omitempty"`

	// Indicates that configuration and container image changes should only be pushed to
	// the first rack of the datacenter
	CanaryUpgrade bool `json:"canaryUpgrade,omitempty"`

	// Turning this option on allows multiple server pods to be created on a k8s worker node.
	// By default the operator creates just one server pod per k8s worker node using k8s
	// podAntiAffinity and requiredDuringSchedulingIgnoredDuringExecution.
	AllowMultipleNodesPerWorker bool `json:"allowMultipleNodesPerWorker,omitempty"`

	// This secret defines the username and password for the Cassandra server superuser.
	// If it is omitted, we will generate a secret instead.
	SuperuserSecretName string `json:"superuserSecretName,omitempty"`

	// The k8s service account to use for the server pods
	ServiceAccount string `json:"serviceAccount,omitempty"`

	// Whether to do a rolling restart at the next opportunity. The operator will set this back
	// to false once the restart is in progress.
	RollingRestartRequested bool `json:"rollingRestartRequested,omitempty"`

	// A map of label keys and values to restrict Cassandra node scheduling to k8s workers
	// with matchiing labels.
	// More info: https://kubernetes.io/docs/concepts/configuration/assign-pod-node/#nodeselector
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// Rack names in this list are set to the latest StatefulSet configuration
	// even if Cassandra nodes are down. Use this to recover from an upgrade that couldn't
	// roll out.
	ForceUpgradeRacks []string `json:"forceUpgradeRacks,omitempty"`

	DseWorkloads *DseWorkloads `json:"dseWorkloads,omitempty"`

	// PodTemplate provides customisation options (labels, annotations, affinity rules, resource requests, and so on) for the cassandra pods
	PodTemplateSpec *corev1.PodTemplateSpec `json:"podTemplateSpec,omitempty"`

	// Cassandra users to bootstrap
	Users []CassandraUser `json:"users,omitempty"`

	AdditionalSeeds []string `json:"additionalSeeds,omitempty"`

	Reaper *ReaperConfig `json:"reaper,omitempty"`
}

type DseWorkloads struct {
	AnalyticsEnabled bool `json:"analyticsEnabled,omitempty"`
	GraphEnabled     bool `json:"graphEnabled,omitempty"`
	SearchEnabled    bool `json:"searchEnabled,omitempty"`
}

type StorageConfig struct {
	CassandraDataVolumeClaimSpec *corev1.PersistentVolumeClaimSpec `json:"cassandraDataVolumeClaimSpec,omitempty"`
}

// GetRacks is a getter for the Rack slice in the spec
// It ensures there is always at least one rack
func (dc *CassandraDatacenter) GetRacks() []Rack {
	if len(dc.Spec.Racks) >= 1 {
		return dc.Spec.Racks
	}

	return []Rack{{
		Name: "default",
	}}
}

// Rack ...
type Rack struct {
	// The rack name
	// +kubebuilder:validation:MinLength=2
	Name string `json:"name"`
	// Zone name to pin the rack, using node affinity
	Zone string `json:"zone,omitempty"`
}

type CassandraNodeStatus struct {
	HostID string `json:"hostID,omitempty"`
}

type CassandraStatusMap map[string]CassandraNodeStatus

type DatacenterConditionType string

const (
	DatacenterReady          DatacenterConditionType = "Ready"
	DatacenterInitialized    DatacenterConditionType = "Initialized"
	DatacenterReplacingNodes DatacenterConditionType = "ReplacingNodes"
	DatacenterScalingUp      DatacenterConditionType = "ScalingUp"
	DatacenterUpdating       DatacenterConditionType = "Updating"
	DatacenterStopped        DatacenterConditionType = "Stopped"
	DatacenterResuming       DatacenterConditionType = "Resuming"
	DatacenterRollingRestart DatacenterConditionType = "RollingRestart"
)

type DatacenterCondition struct {
	Type               DatacenterConditionType `json:"type"`
	Status             corev1.ConditionStatus  `json:"status"`
	LastTransitionTime metav1.Time             `json:"lastTransitionTime,omitempty"`
}

func NewDatacenterCondition(conditionType DatacenterConditionType, status corev1.ConditionStatus) *DatacenterCondition {
	return &DatacenterCondition{
		Type:   conditionType,
		Status: status,
	}
}

// CassandraDatacenterStatus defines the observed state of CassandraDatacenter
// +k8s:openapi-gen=true
type CassandraDatacenterStatus struct {
	Conditions []DatacenterCondition `json:"conditions,omitempty"`

	// Deprecated. Use usersUpserted instead. The timestamp at
	// which CQL superuser credentials were last upserted to the
	// management API
	// +optional
	SuperUserUpserted metav1.Time `json:"superUserUpserted,omitempty"`

	// The timestamp at which managed cassandra users' credentials
	// were last upserted to the management API
	// +optional
	UsersUpserted metav1.Time `json:"usersUpserted,omitempty"`

	// The timestamp when the operator last started a Server node
	// with the management API
	// +optional
	LastServerNodeStarted metav1.Time `json:"lastServerNodeStarted,omitempty"`

	// Last known progress state of the Cassandra Operator
	// +optional
	CassandraOperatorProgress ProgressState `json:"cassandraOperatorProgress,omitempty"`

	// +optional
	LastRollingRestart metav1.Time `json:"lastRollingRestart,omitempty"`

	// +optional
	NodeStatuses CassandraStatusMap `json:"nodeStatuses"`

	// +optional
	NodeReplacements []string `json:"nodeReplacements"`

	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CassandraDatacenter is the Schema for the cassandradatacenters API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=cassandradatacenters,scope=Namespaced,shortName=cassdc;cassdcs
type CassandraDatacenter struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CassandraDatacenterSpec   `json:"spec,omitempty"`
	Status CassandraDatacenterStatus `json:"status,omitempty"`
}

type ManagementApiAuthManualConfig struct {
	ClientSecretName string `json:"clientSecretName"`
	ServerSecretName string `json:"serverSecretName"`
	// +optional
	SkipSecretValidation bool `json:"skipSecretValidation,omitempty"`
}

type ManagementApiAuthInsecureConfig struct {
}

type ManagementApiAuthConfig struct {
	Insecure *ManagementApiAuthInsecureConfig `json:"insecure,omitempty"`
	Manual   *ManagementApiAuthManualConfig   `json:"manual,omitempty"`
	// other strategy configs (e.g. Cert Manager) go here
}

type ReaperConfig struct {
	Enabled bool `json:"enabled,omitempty"`

	Image string `json:"image,omitempty"`

	ImagePullPolicy corev1.PullPolicy `json:"imagePullPolicy,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CassandraDatacenterList contains a list of CassandraDatacenter
type CassandraDatacenterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CassandraDatacenter `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CassandraDatacenter{}, &CassandraDatacenterList{})
}

func (dc *CassandraDatacenter) GetConfigBuilderImage() string {
	var image string
	if dc.Spec.ConfigBuilderImage != "" {
		image = dc.Spec.ConfigBuilderImage
	} else if baseImageOs := os.Getenv(EnvBaseImageOs); baseImageOs != "" {
		image = ubi_defaultConfigBuilderImage
	} else {
		image = defaultConfigBuilderImage
	}
	return image
}

// GetServerImage produces a fully qualified container image to pull
// based on either the version, or an explicitly specified image
//
// In the event that no valid image could be retrieved from the specified version,
// an error is returned.
func (dc *CassandraDatacenter) GetServerImage() (string, error) {
	return makeImage(dc.Spec.ServerType, dc.Spec.ServerVersion, dc.Spec.ServerImage)
}

// makeImage takes the server type/version and image from the spec,
// and returns a docker pullable server container image
// serverVersion should be a semver-like string
// serverImage should be an empty string, or [hostname[:port]/][path/with/repo]:[Server container img tag]
// If serverImage is empty, we attempt to find an appropriate container image based on the serverVersion
// In the event that no image is found, an error is returned
func makeImage(serverType, serverVersion, serverImage string) (string, error) {
	if serverImage == "" {
		return getImageForServerVersion(serverType, serverVersion)
	}
	return serverImage, nil
}

// GetRackLabels ...
func (dc *CassandraDatacenter) GetRackLabels(rackName string) map[string]string {
	labels := map[string]string{
		RackLabel: rackName,
	}

	utils.MergeMap(labels, dc.GetDatacenterLabels())

	return labels
}

func (status *CassandraDatacenterStatus) GetConditionStatus(conditionType DatacenterConditionType) corev1.ConditionStatus {
	for _, condition := range status.Conditions {
		if condition.Type == conditionType {
			return condition.Status
		}
	}
	return corev1.ConditionFalse
}

func (dc *CassandraDatacenter) GetConditionStatus(conditionType DatacenterConditionType) corev1.ConditionStatus {
	return (&dc.Status).GetConditionStatus(conditionType)
}

func (status *CassandraDatacenterStatus) SetCondition(condition DatacenterCondition) {
	conditions := status.Conditions
	added := false
	for i := range status.Conditions {
		if status.Conditions[i].Type == condition.Type {
			status.Conditions[i] = condition
			added = true
		}
	}

	if !added {
		conditions = append(conditions, condition)
	}

	status.Conditions = conditions
}

func (dc *CassandraDatacenter) SetCondition(condition DatacenterCondition) {
	(&dc.Status).SetCondition(condition)
}

// GetDatacenterLabels ...
func (dc *CassandraDatacenter) GetDatacenterLabels() map[string]string {
	labels := map[string]string{
		DatacenterLabel: dc.Name,
	}

	utils.MergeMap(labels, dc.GetClusterLabels())

	return labels
}

// GetClusterLabels returns a new map with the cluster label key and cluster name value
func (dc *CassandraDatacenter) GetClusterLabels() map[string]string {
	return map[string]string{
		ClusterLabel: dc.Spec.ClusterName,
	}
}

func (dc *CassandraDatacenter) GetSeedServiceName() string {
	return dc.Spec.ClusterName + "-seed-service"
}

func (dc *CassandraDatacenter) GetAllPodsServiceName() string {
	return dc.Spec.ClusterName + "-" + dc.Name + "-all-pods-service"
}

func (dc *CassandraDatacenter) GetDatacenterServiceName() string {
	return dc.Spec.ClusterName + "-" + dc.Name + "-service"
}

func (dc *CassandraDatacenter) ShouldGenerateSuperuserSecret() bool {
	return len(dc.Spec.SuperuserSecretName) == 0
}

func (dc *CassandraDatacenter) GetSuperuserSecretNamespacedName() types.NamespacedName {
	name := dc.Spec.ClusterName + "-superuser"
	namespace := dc.ObjectMeta.Namespace
	if len(dc.Spec.SuperuserSecretName) > 0 {
		name = dc.Spec.SuperuserSecretName
	}

	return types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}
}

// GetConfigAsJSON gets a JSON-encoded string suitable for passing to configBuilder
func (dc *CassandraDatacenter) GetConfigAsJSON() (string, error) {

	// We use the cluster seed-service name here for the seed list as it will
	// resolve to the seed nodes. This obviates the need to update the
	// cassandra.yaml whenever the seed nodes change.
	seeds := []string{dc.GetSeedServiceName()}
	seeds = append(seeds, dc.Spec.AdditionalSeeds...)

	graphEnabled := 0
	solrEnabled := 0
	sparkEnabled := 0

	if dc.Spec.ServerType == "dse" && dc.Spec.DseWorkloads != nil {
		if dc.Spec.DseWorkloads.AnalyticsEnabled == true {
			sparkEnabled = 1
		}
		if dc.Spec.DseWorkloads.GraphEnabled == true {
			graphEnabled = 1
		}
		if dc.Spec.DseWorkloads.SearchEnabled == true {
			solrEnabled = 1
		}
	}

	modelValues := serverconfig.GetModelValues(
		seeds,
		dc.Spec.ClusterName,
		dc.Name,
		graphEnabled,
		solrEnabled,
		sparkEnabled)

	var modelBytes []byte

	modelBytes, err := json.Marshal(modelValues)
	if err != nil {
		return "", err
	}

	// Combine the model values with the user-specified values

	modelParsed, err := gabs.ParseJSON([]byte(modelBytes))
	if err != nil {
		return "", errors.Wrap(err, "Model information for CassandraDatacenter resource was not properly configured")
	}

	if dc.Spec.Config != nil {
		configParsed, err := gabs.ParseJSON([]byte(dc.Spec.Config))
		if err != nil {
			return "", errors.Wrap(err, "Error parsing Spec.Config for CassandraDatacenter resource")
		}

		if err := modelParsed.Merge(configParsed); err != nil {
			return "", errors.Wrap(err, "Error merging Spec.Config for CassandraDatacenter resource")
		}
	}

	return modelParsed.String(), nil
}

// GetContainerPorts will return the container ports for the pods in a statefulset based on the provided config
func (dc *CassandraDatacenter) GetContainerPorts() ([]corev1.ContainerPort, error) {
	ports := []corev1.ContainerPort{
		{
			// Note: Port Names cannot be more than 15 characters
			Name:          "native",
			ContainerPort: 9042,
		},
		{
			Name:          "inter-node-msg",
			ContainerPort: 8609,
		},
		{
			Name:          "intra-node",
			ContainerPort: 7000,
		},
		{
			Name:          "tls-intra-node",
			ContainerPort: 7001,
		},
		// jmx-port 7199 was here, seems like we no longer need to expose it
		{
			Name:          "mgmt-api-http",
			ContainerPort: 8080,
		},
	}

	config, err := dc.GetConfigAsJSON()
	if err != nil {
		return nil, err
	}

	var f interface{}
	err = json.Unmarshal([]byte(config), &f)
	if err != nil {
		return nil, err
	}

	m := f.(map[string]interface{})
	promConf := utils.SearchMap(m, "10-write-prom-conf")
	if _, ok := promConf["enabled"]; ok {
		ports = append(ports, corev1.ContainerPort{
			Name:          "prometheus",
			ContainerPort: 9103,
		})
	}

	return ports, nil
}

func SplitRacks(nodeCount, rackCount int) []int {
	nodesPerRack, extraNodes := nodeCount/rackCount, nodeCount%rackCount

	var topology []int

	for rackIdx := 0; rackIdx < rackCount; rackIdx++ {
		nodesForThisRack := nodesPerRack
		if rackIdx < extraNodes {
			nodesForThisRack++
		}
		topology = append(topology, nodesForThisRack)
	}

	return topology
}
