// Package tobiko provides constants and utilities for OpenStack Tobiko testing functionality
package tobiko

import (
	"github.com/openstack-k8s-operators/lib-common/modules/storage"
	corev1 "k8s.io/api/core/v1"
)

const (
	// ServiceName - tobiko service name
	ServiceName = "tobiko"

	// Tobiko is the definition of the tobiko group
	Tobiko storage.PropagationType = "Tobiko"

	// PodRunAsUser is the UID to run the Tobiko pod as
	PodRunAsUser = int64(42495)

	// PodRunAsGroup is the GID to run the Tobiko pod as
	PodRunAsGroup = int64(42495)
)

var (
	// TobikoPropagation is the definition of the Tobiko propagation service
	TobikoPropagation = []storage.PropagationType{Tobiko}

	// PodCapabilities defines the Linux capabilities for Tobiko pods
	PodCapabilities = []corev1.Capability{"NET_ADMIN", "NET_RAW"}
)
