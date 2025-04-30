package horizontest

import (
	"github.com/openstack-k8s-operators/lib-common/modules/storage"
)

const (
	ServiceName = "horizontest"

	// HorizonTest is the definition of the horizontest group
	HorizonTest storage.PropagationType = "HorizonTest"
)

// HorizonTestPropagation is the definition of the HorizonTest propagation service
var HorizonTestPropagation = []storage.PropagationType{HorizonTest}
