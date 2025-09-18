// Package tempest provides constants and utilities for OpenStack Tempest testing functionality
package tempest

import (
	"github.com/openstack-k8s-operators/lib-common/modules/storage"
)

const (
	// ServiceName - tempest service name
	ServiceName = "tempest"

	// Tempest is the definition of the tempest group
	Tempest storage.PropagationType = "Tempest"
)

// TempestPropagation is the definition of the Tempest propagation service
var TempestPropagation = []storage.PropagationType{Tempest}
