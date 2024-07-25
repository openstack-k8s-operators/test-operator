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

type extraConfigmapsMounts struct {
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:validation:Required
	// The name of an existing config map for mounting.
	Name string `json:"name"`

	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:validation:Required
	// Path within the container at which the volume should be mounted.
	MountPath string `json:"mountPath"`

	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:validation:optional
	// +kubebuilder:default=""
	// Config map subpath for mounting, defaults to configmap root.
	SubPath string `json:"subPath"`
}
