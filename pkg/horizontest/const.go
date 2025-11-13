// Package horizontest provides constants and utilities for horizon testing functionality
package horizontest

import (
	"github.com/openstack-k8s-operators/lib-common/modules/storage"
	corev1 "k8s.io/api/core/v1"
)

const (
	// ServiceName is the name of the horizon test service
	ServiceName = "horizontest"

	// HorizonTest is the definition of the horizontest group
	HorizonTest storage.PropagationType = "HorizonTest"

	// PodRunAsUser is the UID to run the HorizonTest pod as
	PodRunAsUser = int64(42455)

	// PodRunAsGroup is the GID to run the HorizonTest pod as
	PodRunAsGroup = int64(42455)
)

var (
	// HorizonTestPropagation is the definition of the HorizonTest propagation service
	HorizonTestPropagation = []storage.PropagationType{HorizonTest}

	// PodCapabilities defines the Linux capabilities for HorizonTest pods
	PodCapabilities = []corev1.Capability{"NET_ADMIN", "NET_RAW"}
)
