// Package ansibletest provides constants and utilities for Ansible-based testing functionality
package ansibletest

import (
	"github.com/openstack-k8s-operators/lib-common/modules/storage"
)

const (
	// ServiceName - ansibleTest service name
	ServiceName = "ansibleTest"

	// AnsibleTest is the definition of the ansibletest group
	AnsibleTest storage.PropagationType = "AnsibleTest"
)

// AnsibleTestPropagation is the definition of the AnsibleTest propagation service
var AnsibleTestPropagation = []storage.PropagationType{AnsibleTest}
