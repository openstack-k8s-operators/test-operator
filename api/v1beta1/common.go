/*
Copyright 2023.

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

package v1beta1

import (
	"github.com/openstack-k8s-operators/lib-common/modules/common/condition"
	corev1 "k8s.io/api/core/v1"
)

type ExtraConfigmapsMounts struct {
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MaxLength:=253
	// The name of an existing config map for mounting.
	Name string `json:"name"`

	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:validation:Required
	// Path within the container at which the volume should be mounted.
	MountPath string `json:"mountPath"`

	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:validation:optional
	// +kubebuilder:default:=""
	// Config map subpath for mounting, defaults to configmap root.
	SubPath string `json:"subPath"`
}

type CommonOptions struct {
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:validation:optional
	// +kubebuilder:default=false
	// +optional
	// Use with caution! This parameter specifies whether test-operator should spawn test
	// pods with allowedPrivilegedEscalation: true, automountServiceAccountToken: true
	// and the default capabilities on top of capabilities that are usually needed
	// by the test pods (NET_ADMIN, NET_RAW). This parameter is deemed insecure
	// but it is needed for certain test-operator functionalities to work properly
	// (e.g.: extraRPMs in Tempest CR, or certain set of tobiko tests).
	Privileged bool `json:"privileged"`

	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:="local-storage"
	// StorageClass used to create any test-operator related PVCs.
	StorageClass string `json:"storageClass"`

	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:=""
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// A SELinuxLevel that should be used for test pods spawned by the test
	// operator.
	SELinuxLevel string `json:"SELinuxLevel"`

	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:=""
	// A URL of a container image that should be used by the test-operator for tests execution.
	ContainerImage string `json:"containerImage"`

	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// BackoffLimit allows to define the maximum number of retried executions (defaults to 0).
	// +kubebuilder:default:=0
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:number"}
	BackoffLimit *int32 `json:"backoffLimit,omitempty"`

	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:validation:Optional
	// Extra configmaps for mounting inside the pod
	ExtraConfigmapsMounts []ExtraConfigmapsMounts `json:"extraConfigmapsMounts,omitempty"`

	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// This value contains a nodeSelector value that is applied to test pods
	// spawned by the test operator.
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// This value contains a toleration that is applied to pods spawned by the
	// test pods that are spawned by the test-operator.
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`
}

type CommonOpenstackConfig struct {
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:default=openstack-config
	// +kubebuilder:validation:Optional
	// OpenStackConfigMap is the name of the ConfigMap containing the clouds.yaml
	OpenStackConfigMap string `json:"openStackConfigMap"`

	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:default=openstack-config-secret
	// +kubebuilder:validation:Optional
	// OpenStackConfigSecret is the name of the Secret containing the secure.yaml
	OpenStackConfigSecret string `json:"openStackConfigSecret"`
}

// CommonTestStatus defines the observed state of the controller
type CommonTestStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Map of hashes to track e.g. job status
	Hash map[string]string `json:"hash,omitempty"`

	// Conditions
	Conditions condition.Conditions `json:"conditions,omitempty" optional:"true"`

	// NetworkAttachments status of the deployment pods
	NetworkAttachments map[string][]string `json:"networkAttachments,omitempty"`
}

type WorkflowCommonParameters struct {
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:validation:optional
	// +optional
	// Use with caution! This parameter specifies whether test-operator should spawn test
	// pods with allowedPrivilegedEscalation: true and the default capabilities on
	// top of capabilities that are usually needed by the test pods (NET_ADMIN, NET_RAW).
	// This parameter is deemed insecure but it is needed for certain test-operator
	// functionalities to work properly (e.g.: extraRPMs in Tempest CR, or certain set
	// of tobiko tests).
	Privileged *bool `json:"privileged,omitempty"`

	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:="local-storage"
	// StorageClass used to create any test-operator related PVCs.
	StorageClass *string `json:"storageClass"`

	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:validation:Optional
	// +optional
	// A SELinuxLevel that should be used for test pods spawned by the test
	// operator.
	SELinuxLevel *string `json:"SELinuxLevel,omitempty"`

	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:=""
	// A URL of a container image that should be used by the test-operator for tests execution.
	ContainerImage string `json:"containerImage"`

	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// BackoffLimit allows to define the maximum number of retried executions (defaults to 0).
	// +kubebuilder:default:=0
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:number"}
	BackoffLimit *int32 `json:"backoffLimit,omitempty"`

	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:validation:Optional
	// Extra configmaps for mounting inside the pod
	ExtraConfigmapsMounts *[]ExtraConfigmapsMounts `json:"extraConfigmapsMounts,omitempty"`

	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// This value contains a nodeSelector value that is applied to test pods
	// spawned by the test operator.
	NodeSelector *map[string]string `json:"nodeSelector,omitempty"`

	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// This value contains a toleration that is applied to pods spawned by the
	// test pods that are spawned by the test-operator.
	Tolerations *[]corev1.Toleration `json:"tolerations,omitempty"`
}
