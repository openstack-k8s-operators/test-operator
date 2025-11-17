// Package tempest provides constants and utilities for OpenStack Tempest testing functionality
package tempest

import (
	"github.com/openstack-k8s-operators/lib-common/modules/storage"
	corev1 "k8s.io/api/core/v1"
)

const (
	// ServiceName - tempest service name
	ServiceName = "tempest"

	// Tempest is the definition of the tempest group
	Tempest storage.PropagationType = "Tempest"

	// PodRunAsUser is the UID to run the Tempest pod as
	PodRunAsUser = int64(42480)

	// PodRunAsGroup is the GID to run the Tempest pod as
	PodRunAsGroup = int64(42480)
)

var (
	// TempestPropagation is the definition of the Tempest propagation service
	TempestPropagation = []storage.PropagationType{Tempest}

	// PodCapabilities defines the Linux capabilities for Tempest pods
	PodCapabilities = []corev1.Capability{}
)
