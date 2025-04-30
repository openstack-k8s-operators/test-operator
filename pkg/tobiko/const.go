package tobiko

import (
	"github.com/openstack-k8s-operators/lib-common/modules/storage"
)

const (
	// ServiceName - tempest service name
	ServiceName = "tobiko"

	// Tobiko is the definition of the tobiko group
	Tobiko storage.PropagationType = "Tobiko"
)

// TobikoPropagation is the definition of the Tobiko propagation service
var TobikoPropagation = []storage.PropagationType{Tobiko}
