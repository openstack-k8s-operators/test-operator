.. _custom-resources-used-by-the-test-operator:

=============================
CRs Used By the test-operator
=============================

This file contains definitions of all custom resources (CRs) that are accepted by
the test-operator. Currently, there are four CRs accepted by the test-operator:

* :ref:`tempest-custom-resource`

* :ref:`tobiko-custom-resource`

* :ref:`horizontest-custom-resource`

* :ref:`ansibletest-custom-resource`


.. _tempest-custom-resource:

Tempest Custom Resource
=======================

.. literalinclude:: ../../config/samples/test_v1beta1_tempest.yaml
   :language: yaml


.. _tobiko-custom-resource:

Tobiko Custom Resource
======================

.. literalinclude:: ../../config/samples/test_v1beta1_tobiko.yaml
   :language: yaml


.. _horizontest-custom-resource:

HorizonTest Custom Resource
===========================

.. literalinclude:: ../../config/samples/test_v1beta1_horizontest.yaml
   :language: yaml


.. _ansibletest-custom-resource:

AnsibleTest Custom Resource
===========================

.. literalinclude:: ../../config/samples/test_v1beta1_ansibletest.yaml
   :language: yaml


.. _git-branch-selection-for-ansibletest:

Git Branch Selection for AnsibleTest
====================================
AnsibleTest runs Ansible playbooks from a git repository inside a test pod.
By default, the pod clones the repository and uses its default branch. The
git branch selection feature allows you to specify which branch, tag, or ref
should be checked out before the playbook runs.

Functionality
-------------

The feature is controlled by two related fields in the AnsibleTest CR spec:

:code:`ansibleGitRepo` **(required)**

The URL of the git repository to clone into the test pod. This must be a
valid URI accessible from within the cluster.

:code:`ansibleGitBranch` **(optional, no default)**

The branch, tag, or git ref to check out after cloning. When omitted or
left empty, the repository's default branch is used.

The operator passes both values to the test pod as environment variables.
The container's entrypoint script handles the clone and checkout.

Basic Usage
-----------

Run playbooks from a specific branch:

.. code-block:: yaml

   apiVersion: test.openstack.org/v1beta1
   kind: AnsibleTest
   metadata:
     name: release-validation
     namespace: openstack
   spec:
     ansibleGitRepo: "https://github.com/myorg/openstack-tests"
     ansibleGitBranch: "release-2.1"
     ansiblePlaybookPath: "playbooks/validate.yaml"

The pod clones the repository, checks out :code:`release-2.1`, then runs the
playbook.

Run playbooks from a specific tag:

.. code-block:: yaml

   spec:
     ansibleGitRepo: "https://github.com/myorg/openstack-tests"
     ansibleGitBranch: "v3.0.0"
     ansiblePlaybookPath: "playbooks/regression.yaml"

The :code:`ansibleGitBranch` field accepts any valid git ref — branch names,
tag names, or commit SHAs all work.

.. note::
   If :code:`ansibleGitBranch` is omitted or set to an empty string, the
   repository's default branch is used.

.. note::
   In workflows, a step that provides an empty :code:`ansibleGitBranch` ("")
   will not override the base spec's value, because the merge logic
   skips empty strings. To explicitly use the default branch in a workflow
   step when the base spec sets a branch, remove :code:`ansibleGitBranch`
   from the base spec and set it only on the steps that need it.


.. _parallel-execution:

Parallel Execution
==================
By default, test-operator runs pods serially. The reason for this is to prevent
collisions between tests (e.g., two tests from two different frameworks modifying
the same resource). So, for example, if you run the following two commands:

.. code-block:: bash

   oc apply -f crd1.yaml
   oc apply -f crd2.yaml

where :code:`crd1.yaml` and :code:`crd2.yaml` are, let's say, two Tobiko CRs, then
you will first see a test pod executing tests defined by :code:`crd1.yaml`. After
the test pod finishes, you will see a second test pod that will be executing tests
specified in :code:`crd2.yaml`.

If you want to run multiple test pods at the same time, then you need to set
:code:`parallel: true` in the :code:`Spec` section in both :code:`crd1.yaml`
and :code:`crd2.yaml`:

.. code-block:: yaml

   ---
   apiVersion: test.openstack.org/v1beta1
   kind: Tobiko
   metadata:
     name: crd1
     namespace: openstack
   spec:
     parallel: true # <-- HERE
     testenv: py3
   ---
   apiVersion: test.openstack.org/v1beta1
   kind: Tobiko
   metadata:
     name: crd1
     namespace: openstack
   spec:
     parallel: true # <-- HERE
     testenv: py3

.. _workflow:

Workflow Section
================
The workflow section enables the spawning of multiple test pods at the same
time. For example, in the Tempest CR shown below, two test pods are spawned,
each corresponding to a different step. Each step inherits a configuration
that is specified outside the workflow section. In individual steps, you can
overwrite values specified in the :code:`tempestRun` and
:code:`tempestconfRun` sections.

.. code-block:: yaml

  ---
  apiVersion: test.openstack.org/v1beta1
  kind: Tempest
  metadata:
    name: tempest-tests
    namespace: openstack
  spec:
    containerImage: ""
    # parallel: true # <-- Uncomment for parallel execution
    tempestRun:
    includeList: |
      tempest.api.identity.v3.*
    concurrency: 8
    tempestconfRun:
    workflow:
      - stepName: first-step
        tempestRun:
          includeList: |
            tempest.api.*
      - stepName: second-step
        tempestRun:
          includeList: |
            neutron_tempest_plugin.*

By default, test pods are executed sequentially. To enable parallel
execution of test pods, you need to set :code:`parallel: true` in the
:code:`spec` section.

CRs that can use the workflow section:

* :ref:`tempest-custom-resource`

* :ref:`tobiko-custom-resource`

* :ref:`ansibletest-custom-resource`

.. _configurable-openstack-config:

Configurable OpenStack Config
=============================
When the test-operator runs tests (Tempest, Tobiko, AnsibleTest, HorizonTest),
the test pods need to authenticate against an OpenStack cloud deployment. The
operator expects two manifest files to exist inside the cluster which are then
mounted into every test pod:

* :code:`clouds.yaml` — defines where the cloud is (endpoint URLs, project
  name, username). Stored as a **ConfigMap**.

* :code:`secure.yaml` — defines how to log in (passwords, tokens). Stored as
  a **Secret**.

The configurable OpenStack config feature makes the names of these resources
user-settable fields on every test CR:

.. code-block:: yaml

   spec:
     openStackConfigMap: "openstack-config"           # default
     openStackConfigSecret: "openstack-config-secret"  # default

Both fields are optional. If you do not set them, the defaults apply. If set,
the operator uses your custom names instead.

Password Injection for HorizonTest and Tobiko
----------------------------------------------

Some OpenStack deployments provide a :code:`clouds.yaml` without a password —
instead the password is stored in the separate :code:`secure.yaml`. The
HorizonTest and Tobiko testing frameworks expect the password to appear inline
in :code:`clouds.yaml`.

To handle this, the controller creates a derived ConfigMap called
:code:`test-operator-clouds-config`. It reads the original ConfigMap and, if
the password field is missing, injects a default placeholder. The patched
:code:`clouds.yaml` is written into the new ConfigMap, and the test pod mounts
this derived ConfigMap instead of the original.

The controller handles the transformation transparently — the user only needs
to point :code:`openStackConfigMap` to their original ConfigMap without
worrying about the injected password.

.. note::
   The password injection step only applies to HorizonTest and Tobiko. Tempest
   and AnsibleTest mount the original ConfigMap and Secret directly.

.. _network-attachments:

Network Attachments
===================
OpenStack deployments distribute services across multiple isolated networks.
These networks are deliberately separated for security and traffic management.
The test-operator pod must be attached to the same networks where those
resources live to cover varying scenarios such as creating a virtual machine,
assigning a floating IP, and confirming SSH access.

The :code:`networkAttachments` field allows users to specify additional
networks that the test pod should be connected to, using Kubernetes
:code:`NetworkAttachmentDefinition` (NAD) resources provided by the Multus CNI
plugin.

.. note::
   NADs must exist in the same namespace as the test CR.

Supported Test Types
--------------------

Not every test framework in the test-operator requires multi-network access.
The feature is enabled per CRD based on what each framework does at the
network level:

* **Tempest** - Supported. It exercises the full OpenStack API surface. Many
  Tempest tests go beyond API calls, creating infrastructure and then
  validating that infrastructure by testing network connectivity.

* **Tobiko** - Supported. It creates long-lived OpenStack resources, runs
  sustained reachability checks, and deliberately disrupts services to validate
  recovery.

* **HorizonTest** - Not supported.

* **AnsibleTest** - Not supported.

The IP addresses assigned to each interface are visible in the CR's status:

.. code-block:: yaml

   status:
     networkAttachments:
       ctlplane:
         - 192.168.1.42
       internalapi:
         - 172.17.0.42
     conditions:
       - type: NetworkAttachmentsReady
         status: "True"
         message: NetworkAttachments ready

Usage
-----

Prerequisites:

* Multus CNI is installed on the cluster.

* The required :code:`NetworkAttachmentDefinition` resources exist in the
  target namespace.

* Both are managed by cluster administrators, not by the test-operator.

Basic example:

.. code-block:: yaml

   apiVersion: test.openstack.org/v1beta1
   kind: Tempest
   metadata:
     name: tempest-test-nad
     namespace: openstack
   spec:
     networkAttachments:
       - ctlplane
     containerImage: quay.io/podified-antelope-centos9/openstack-tempest:current-podified

The test pod will receive two network interfaces: the default pod network
interface, plus one for :code:`ctlplane`.

Per-Workflow-Step Configuration
^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^

When using workflows, each step can specify its own set of network
attachments. Steps that do not override the field inherit from the base spec:

.. code-block:: yaml

   apiVersion: test.openstack.org/v1beta1
   kind: Tempest
   metadata:
     name: tempest-workflow
     namespace: openstack
   spec:
     networkAttachments:
       - ctlplane
     workflow:
       - stepName: firstStep-tests
         networkAttachments:
           - ctlplane
           - external
       - stepName: secondStep-tests

In this example, the :code:`firstStep-tests` step attaches to both
:code:`ctlplane` and :code:`external`. The :code:`secondStep-tests` step
inherits the base spec and attaches only to :code:`ctlplane`.

Omitting the Field
^^^^^^^^^^^^^^^^^^

If :code:`networkAttachments` is not specified, the test pod receives only the
default Kubernetes pod network. The Multus dependency is not required. The
:code:`NetworkAttachmentsReady` condition will also not appear in the CR's status.

.. _nodeselector-and-tolerations:

NodeSelector and Tolerations
============================
The test-operator creates pods to run OpenStack tests. By default, the
Kubernetes scheduler places these pods on any available node based on resource
availability. In many OpenStack test environments this is not desirable as
nodes may be partitioned by role, reserved for specific workloads, or
protected by taints that reject pods without explicit permission.

:code:`nodeSelector` and :code:`tolerations` give users control over which
nodes the test-operator schedules its pods on. Both fields are available on
all four test CRDs (Tempest, Tobiko, AnsibleTest, and HorizonTest). When
omitted, the scheduler uses its default placement logic.

Behaviour Per Test
------------------

* **Tempest**: Tempest test suites can run hundreds of tests in parallel,
  consuming significant CPU and memory. Use :code:`nodeSelector` to target
  nodes labeled for testing and :code:`tolerations` to access tainted compute
  nodes reserved for tests, keeping Tempest from interfering with the
  deployment it is testing. Workflow support means different Tempest steps can
  target different nodes.

* **Tobiko**: Tobiko runs scenario tests and fault injection over extended
  periods. It creates virtual machines, monitors network reachability, and
  deliberately disrupts services. These long-running pods benefit from being
  placed on stable, dedicated nodes where they are less likely to be preempted
  or affected by resource contention. Tobiko also supports per-workflow-step
  scheduling overrides.

* **AnsibleTest**: AnsibleTest supports workflow steps with per-step
  scheduling overrides, allowing different playbooks to target different nodes
  if needed. A common pattern is running privileged Ansible tasks on nodes
  that permit elevated security contexts while keeping non-privileged tasks on
  standard nodes.

* **HorizonTest**: Scheduling Horizon tests onto appropriately labeled nodes
  avoids failures caused by node-level incompatibilities. HorizonTest does not
  support workflows, so there are no per-step overrides. The scheduling
  configuration applies to the single test pod.

Basic Usage
-----------

.. code-block:: yaml

   apiVersion: test.openstack.org/v1beta1
   kind: AnsibleTest
   metadata:
     name: ansible-validate
     namespace: openstack
   spec:
     nodeSelector:
       testoperator: "true"
     tolerations:
       - key: "test"
         operator: "Equal"
         value: "true"
         effect: "NoSchedule"
     containerImage: quay.io/podified-antelope-centos9/openstack-ansible-tests:current-podified

The toleration grants permission to run on the tainted node. The
:code:`nodeSelector` ensures the pod only runs on nodes with the matching
label.

.. _security-hardening:

Security Hardening
==================
The test-operator applies a hardened security profile to all test pods by
default. The following measures restrict what containers can do at the kernel,
filesystem, and capability level.

Seccomp Profile
---------------

A :code:`RuntimeDefault` seccomp profile is enabled on every test pod,
restricting the container to only the system calls expected during normal
operation. This is a kernel-level filter that blocks unusual syscalls that
test pods have no reason to make.

Privilege Escalation
--------------------

:code:`allowPrivilegeEscalation` is set to :code:`false` by default,
preventing processes inside the container from gaining more privileges than
their parent. Two unnecessary host mounts (:code:`/etc/localtime` and
:code:`/etc/machine-id`) were also removed — they required :code:`hostPath`
volumes, which are incompatible with a non-privileged security profile.

Read-Only Root Filesystem
-------------------------

Test pods run with :code:`readOnlyRootFilesystem: true` by default. A
writable root filesystem allows a compromised container to modify its own
binaries or write to unexpected locations. The only case where a writable
filesystem is needed is when Tempest installs additional RPMs at runtime via
the :code:`extraRPMs` feature, which requires :code:`privileged: true`.

SELinux Level Default
---------------------

The :code:`SELinuxLevel` field has no default value. Setting an SELinux level
on a pod requires the privileged Security Context Constraints (SCC) in
OpenShift, so leaving a default would cause pods to unexpectedly require
elevated SCCs even when the user had not opted into privileged mode. Users set
this field explicitly when needed, typically when running privileged workflows
that write to shared PVCs.

Capability Dropping
-------------------

The :code:`NET_ADMIN` and :code:`NET_RAW` capabilities are only added when
:code:`privileged: true` is set on the CR spec. When :code:`privileged: false`
(the default), all capabilities are explicitly dropped. These capabilities
allow raw socket access and network configuration changes, which are needed
for tools like :code:`tcpdump` in Tobiko but unnecessary for most test runs.
See :ref:`privileged-mode-and-capabilities` for more comprehensive
information.

.. _privileged-mode-and-capabilities:

Privileged Mode and Capabilities
================================
:code:`NET_ADMIN` and :code:`NET_RAW` are the two capabilities that make
privileged mode useful for test pods.

:code:`NET_ADMIN` allows the process to modify network configuration,
changing routing tables, configuring network interfaces, setting up traffic
control rules, and modifying firewall rules. Tempest tests that exercise
neutron functionality may need this to inspect or manipulate the network stack
from within the test pod.

:code:`NET_RAW` allows the process to create raw network sockets and craft
arbitrary packets. This is required by tools like :code:`ping` and
:code:`tcpdump`, consumed by Tempest network connectivity tests.

Behaviour Per Framework
-----------------------

* **Tempest**: Network plugin tests (:code:`neutron-tempest-plugin`) may
  verify connectivity by pinging VMs, inspecting routing tables, or running
  network diagnostic tools from within the test pod. Compute tests that
  validate metadata service access or security group enforcement may also need
  raw socket access.

* **Tobiko**: Tobiko's fault injection and scenario tests routinely use
  :code:`tcpdump` to capture traffic, :code:`ping` to verify reachability,
  and network manipulation tools to simulate failures.

* **AnsibleTest**: Most Ansible tests may not need these capabilities, but
  the option exists for playbooks that perform network-level validation.

* **HorizonTest**: Horizon tests do not typically require raw socket access,
  as interaction is driven via a web browser.

.. _image-creation-timeout:

Image Creation Timeout
======================
Tempest tests may require specific virtual machine images to exist in
OpenStack's image service (Glance) before the tests can run. The
test-operator's :code:`extraImages` feature handles this by downloading image
files into the test pod and uploading them to Glance as part of the test
setup.

The :code:`imageCreationTimeout` field controls how long the test pod waits
for an uploaded image to transition from :code:`queued` or :code:`saving` to
:code:`active` in Glance. If the image does not become active within this
window, the test setup fails.

.. note::
   This field is specific to the Tempest CRD. It does not apply to Tobiko,
   AnsibleTest, or HorizonTest.

Functionality
-------------

The :code:`imageCreationTimeout` field gives users explicit control over the
image upload wait period, making failures predictable and the root cause
visible.

The field is part of the :code:`extraImages` configuration on the Tempest CR.
Each entry in :code:`extraImages` describes an image to download and upload to
Glance. The timeout applies per image. The operator passes the value to the
Tempest container as the environment variable
:code:`TEMPEST_EXTRA_IMAGES_CREATE_TIMEOUT`. The script running inside the
container uses this value when polling Glance for the image status after
upload. If the image does not reach :code:`active` within the specified number
of seconds, the script exits with an error and the test pod fails.

.. note::
   The default value is 300 seconds (5 minutes).

.. _tempest-cleanup:

Tempest Cleanup
===============
When Tempest runs tests, complexity presents itself in the cleanup stage —
bugs introduced during teardown logic such as timeouts interrupting cleanup
and parallel tests that create ordering problems where one test's teardown
interferes with another's.

Tempest includes a built-in :code:`tempest cleanup` command specifically for
this problem. It compares the current state of the OpenStack deployment
against a saved snapshot of what existed before the tests ran and deletes
anything that was added during the test execution. The test-operator's
:code:`cleanup` field activates this behaviour as a post-test phase within
the same pod.

The :code:`tempest cleanup` command targets resources across all OpenStack
services that Tempest tests interact with.

.. note::
   This field is specific to the Tempest CRD. It does not apply to Tobiko,
   AnsibleTest, or HorizonTest.

Basic Usage
-----------

The :code:`cleanup` field is a boolean on the Tempest CR spec, defaulting to
:code:`false`:

.. code-block:: yaml

   apiVersion: test.openstack.org/v1beta1
   kind: Tempest
   metadata:
     name: tempest-test-cleanup
     namespace: openstack
   spec:
     cleanup: true
     containerImage: quay.io/podified-antelope-centos9/openstack-tempest:current-podified

The operator passes this as the environment variable
:code:`TEMPEST_CLEANUP=true` to the test pod. The entrypoint script inside
the container checks this variable and, if set, runs :code:`tempest cleanup`
after the test suite finishes. The cleanup phase runs regardless of whether
the tests passed or failed,this is by design, as failed tests are the most
common source of orphaned resources.

.. note::
   Cleanup is a global setting on the Tempest CR. It cannot be overridden per
   workflow step. When enabled, it applies after each step's test execution.

ExtraMounts parameter
=====================
To correctly use the :code:`ExtraMounts` parameter, follow these steps:

1. Set the :code:`propagation` field. Set this field based on the test
scope (e.g., Tempest, Tobiko) to control where the mount is applied.

2. Set the :code:`volumes` field. Define the list of volume sources
to be mounted. The name assigned here is later referenced in the
:code:`mounts` field.

3. Set the :code:`mounts` field. Specify where each volume should be
mounted in the Pod. Each entry should include the name of a volume
from the :code:`volumes` field and the target mount path.

Example test of using the :code:`ExtraMounts` parameter:

.. code-block:: yaml

  extraMounts:
      - name: v1
        region: r1
        extraVol:
          - propagation:
            - Tempest
            extraVolType: Ceph
            volumes:
            - name: ceph
              secret:
                secretName: <existing-secret>
            mounts:
            - name: ceph
              mountPath: "/etc/ceph"
              readOnly: true
