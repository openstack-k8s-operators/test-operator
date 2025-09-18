// Package horizontest provides constants and utilities for horizon testing functionality
package horizontest

import (
	"github.com/openstack-k8s-operators/lib-common/modules/storage"
)

const (
	// ServiceName is the name of the horizon test service
	ServiceName = "horizontest"

	// HorizonTest is the definition of the horizontest group
	HorizonTest storage.PropagationType = "HorizonTest"
)

// HorizonTestPropagation is the definition of the HorizonTest propagation service
var HorizonTestPropagation = []storage.PropagationType{HorizonTest}
