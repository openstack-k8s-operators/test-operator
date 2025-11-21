// Package ansibletest provides constants and utilities for Ansible-based testing functionality
package ansibletest

import (
	"github.com/openstack-k8s-operators/lib-common/modules/storage"
	corev1 "k8s.io/api/core/v1"
)

const (
	// ServiceName - ansibleTest service name
	ServiceName = "ansibleTest"

	// AnsibleTest is the definition of the ansibletest group
	AnsibleTest storage.PropagationType = "AnsibleTest"

	// PodRunAsUser is the UID to run the AnsibleTest pod as
	PodRunAsUser = int64(227)

	// PodRunAsGroup is the GID to run the AnsibleTest pod as
	PodRunAsGroup = int64(227)
)

var (
	// AnsibleTestPropagation is the definition of the Ansible Test propagation service
	AnsibleTestPropagation = []storage.PropagationType{AnsibleTest}

	// PodCapabilities defines the Linux capabilities for AnsibleTest pods
	PodCapabilities = []corev1.Capability{"NET_ADMIN", "NET_RAW"}
)
