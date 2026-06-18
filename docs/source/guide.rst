Run Tests via Test Operator
===========================

.. note::
   Before you proceed with this section of the documentation, please make sure
   that you have read the :ref:`prerequisites <prerequisites>`.

This guide describes:

* How to **install/uninstall** the operator?

  * :ref:`running-operator-olm`

  * :ref:`running-operator-locally`

  * :ref:`uninstalling-operator`

* How to **run tests** via the operator?

  * :ref:`executing-tests`

  * :ref:`getting-logs`

If you want to get your hands on the test-operator quickly, then follow these two
sections: :ref:`running-operator-olm` and :ref:`executing-tests`.

.. note::
  If you prefer visual guides, you can check out
  `Test Operator Tempest Guide <https://www.youtube.com/watch?v=nz72z5goEP8>`_ video.

.. _running-operator-olm:

Running Test Operator Using the Operator Lifecycle Manager (OLM)
----------------------------------------------------------------

The first option of how to start the operator is by running the pre-built operator image
stored in the `openstack-operator-index <https://quay.io/repository/openstack-k8s-operators/openstack-operator-index>`_
using the OLM.

Follow these steps to install the operator in the :code:`openstack-operators`
project.

1. Create :code:`Subscription`

.. code-block:: yaml

   cat subscription.yaml
   ---
   apiVersion: operators.coreos.com/v1alpha1
   kind: Subscription
   metadata:
     name: test-operator
     namespace: openstack-operators
   spec:
     name: test-operator
     source: openstack-operator-index
     sourceNamespace: openstack-operators

2. Apply :code:`subscription.yaml`

.. code-block:: bash

   oc apply -f subscription.yaml

3. Wait for the :code:`test-operator-controller-manager` pod to successfully
   spawn. Once you see the pod running, you can start communicating with the
   operator using the CRs understood by the test-operator (see
   :ref:`custom-resources-used-by-the-test-operator`). For more information
   about how to run tests via the test-operator, refer to the :ref:`executing-tests`
   section.

.. code-block:: bash

   oc get pods -n openstack-operators
   ...
   test-operator-controller-manager-6c9994847c-6jwn5                 2/2     Running     0              20s
   ...

.. _running-operator-locally:

Running Test Operator Locally Outside the Cluster
-------------------------------------------------
This is an easy way to experiment with the operator during development
of a new feature.

.. code-block:: bash

    ENABLE_WEBHOOKS=false make install run

Note that after running the following command, you will need to switch to
another terminal unless you run it in the background.

Now that test-operator is automatically deployed in the podified environment,
running changes locally may be interrupted by the :code:`test-operator-controller-manager`
pod, which runs by default. To prevent errors, disable test-operator before
testing local changes by following these steps.

1. Scale down the openstack-operator deployment.
   This step is necessary because if you only delete the controller managers,
   the reconcile loop will redeploy them.

.. code-block:: bash

   oc scale deployment openstack-operator-controller-operator -n openstack-operators --replicas=0

2. Delete the openstack-operator and test-operator controller managers

.. code-block:: bash

   oc delete deployment openstack-operator-controller-manager -n openstack-operators
   oc delete deployment test-operator-controller-manager -n openstack-operators

3. Check deletion of :code:`test-operator-controller-manager`

.. code-block:: bash

   oc get deployment -n openstack-operators | grep test-operator-controller-manager

.. note::
   If you want to revert the changes, simply scale the
   openstack-operator controller back to one replica using command

   .. code-block:: bash

      oc scale deployment openstack-operator-controller-operator -n openstack-operators --replicas=1

.. _uninstalling-operator:

Uninstalling Operator
---------------------

If you installed the operator by following the steps in the
:ref:`running-operator-olm` section, then this section can come in handy. You
might need to uninstall the operator when:

* you encountered issues during the installation process or when

* you want to be sure that you are using the latest version of the operator.

Please make sure that you follow the order of the steps:

1. Remove all instances of the CRDs supported by the test-operator (:code:`Tempest`,
   :code:`Tobiko`, ...)

.. code-block:: bash

   oc delete tempest --all
   oc delete tobiko --all
   oc delete horizontests --all
   oc delete ansibletests --all

2. Remove the :code:`crd`

.. code-block:: bash

   oc delete crd/tempests.test.openstack.org
   oc delete crd/tobikoes.test.openstack.org
   oc delete crd/ansibletests.test.openstack.org
   oc delete crd/horizontests.test.openstack.org

3. Remove the :code:`subscription` you created during
   :ref:`the installation <running-operator-olm>`.

.. code-block:: bash

   oc delete subscription/test-operator -n openstack-operators

4. Remove the :code:`csv`

.. code-block:: bash

   oc delete clusterserviceversion.operators.coreos.com/test-operator.v0.0.1 -n openstack-operators

5. Remove test-operator related installplan (replace :code:`XXXXX` with value obtained
   with the first command :code:`oc get installplans`)

.. code-block:: bash

   oc get installplans -n openstack-operators | grep "test-operator"
   oc delete installplan install-XXXXX -n openstack-operators


6. Remove the :code:`operator`. It is possible that if you executed
   the previous commands too quickly, then you will need to execute this
   command twice.

.. code-block:: bash

   oc delete operator/test-operator.openstack-operators

7. Check that there are no test-operator related resources hanging. This step
   is not required.

.. code-block:: bash

   oc get olm -n openstack-operators

.. note::
   It might happen that by changing the order of the uninstallation steps,
   you encounter a situation when you will not be able to delete the
   :code:`crd`. In such a case, try to delete the :code:`finalizers:`
   section in the output of the :code:`oc edit tempest/tempest-tests`.


.. _executing-tests:

Executing Tests
---------------

Once you have the test operator running, then you can apply a custom resource accepted
by the test-operator to start the testing. Currently, four types of custom
resources are being accepted by the test-operator (see
:ref:`custom-resources-used-by-the-test-operator` section):

* :ref:`tempest-custom-resource`

* :ref:`tobiko-custom-resource`

* :ref:`horizontest-custom-resource`

* :ref:`ansibletest-custom-resource`


1. Create a manifest for custom resource accepted by the test-operator
   (:ref:`custom-resources-used-by-the-test-operator` section).

2. Apply the manifest. Either go with the default one, the command below, or
   replace the path with a manifest created in the first step.

.. code-block:: bash

    oc apply -f config/samples/test_v1beta1_tempest.yaml

3. Verify that the pod executing the tests is running. It might take a couple
   of seconds for the test pod to spawn. Also, note that by default, the test-operator
   allows only one test pod to be running at the same time (read
   :ref:`parallel-execution`). If you defined your own custom resource in the first step,
   then your test pod will be named according to the :code:`name` value stored in the
   metadata section.

.. code-block:: bash

    oc get pods | grep tempest

You should see a pod with a name like :code:`tempest-tests-xxxxx`.

4. Investigate the stdout of the test-pod:

.. code-block:: bash

    oc logs <name of the pod>

Read :ref:`getting-logs` section if you want to see logs and artifacts
produced during the testing.

.. _service-config-ready-condition:

ServiceConfigReady Condition
----------------------------
The :code:`ServiceConfigReady` condition tells you whether the operator has
successfully generated the internal ConfigMaps that a test pod needs to run.
This condition covers operator-generated ConfigMaps such as environment variable
bundles, template parameters,and for HorizonTest and Tobiko tests, the derived
:code:`test-operator-clouds-config` ConfigMap with the patched password (see
:ref:`configurable-openstack-config`).

For example, running :code:`oc describe tempest my-test -n openstack` shows:

.. code-block:: text

   Conditions:
     Type                          Status  Reason      Message
     ----                          ------  ------      -------
     InputReady                    True    Ready       Input validation passed
     ServiceConfigReady            True    Ready       Service config ready
     DeploymentReady               False   Requested   Deployment is running
     NetworkAttachmentsReady       True    Ready       Network attachments ready
     Ready                         False   Requested   Deployment is running

Diagnostic Signals
^^^^^^^^^^^^^^^^^^

:code:`ServiceConfigReady = Unknown`

The operator has not reached this step yet (still initializing or validating
inputs).

:code:`ServiceConfigReady = True`

Config generation succeeded; the operator moved on to creating the pod.

:code:`ServiceConfigReady = False`

Config generation failed. The accompanying message describes the exact
failure.

What the Controller Generates
^^^^^^^^^^^^^^^^^^^^^^^^^^^^^

The controller calls a resource-specific callback
(:code:`GenerateServiceConfigMaps`) that produces different ConfigMaps for each
resource:

* **Tempest**: ConfigMaps with environment variables for test concurrency,
  external plugin deployment information, worker counts, and custom Tempest
  configuration.

* **Tobiko**: ConfigMaps with Tobiko-specific environment variables (test
  suites, prevent-create flags, debug settings).

* **HorizonTest**: Calls :code:`EnsureCloudsConfigMapExists` to create the
  derived :code:`test-operator-clouds-config` ConfigMap, plus an additional
  environment variable ConfigMap.

Inspecting the Condition
^^^^^^^^^^^^^^^^^^^^^^^^

Quick table view:

.. code-block:: bash

   oc get tempest -n openstack

Shows :code:`Status` and :code:`Message` columns from the first condition.

Full condition details:

.. code-block:: bash

   oc get tempest my-test -n openstack -o jsonpath='{.status.conditions}' | jq

Filter for :code:`ServiceConfigReady` specifically:

.. code-block:: bash

   oc get tempest my-test -n openstack \
     -o jsonpath='{.status.conditions[?(@.type=="ServiceConfigReady")]}'

Human-readable output:

.. code-block:: bash

   oc describe tempest my-test -n openstack

Look for the **Conditions:** section in the output.

.. _tempest-rerun-failed-tests:

Tempest Re-run Failed Tests
---------------------------
In deployments, some tests occasionally fail not due to bug failures, but
because of temporary conditions: service overload, network
routes taking longer than expected to converge, or a virtual machine that did not reach
its expected state within a timeout. These intermittent failures are commonly
called **flaky tests**.

The re-run feature within the Test Operator addresses this problem. When enabled, the Tempest
pod automatically re-executes any tests that failed during the initial run. This occurs inside
the same pod, during the same execution without the need of manual intervention and thus, no
second pod is created.

Functionality
^^^^^^^^^^^^^

The feature is controlled by two fields in the Tempest CR spec:

:code:`rerunFailedTests` **(default: false)**

When set to :code:`true`, the Tempest container will perform a second run after the
initial execution completes, targeting only the tests that failed. If all tests
passed on the first run, no re-run shall occur.

:code:`rerunOverrideStatus` **(default: false)**

Controls what the pod reports as its final result when a re-run happens:

* :code:`false` - The pod exits with the result of the original run. If
  all previously failed tests pass on re-run, the pod still reports failure.
  The re-run results are available in the logs for investigation.

* :code:`true` - The pod exits with the result of the re-run. If all failed
  tests pass on the second attempt, the pod reports success on the consecutive run.

CI Pipeline Gating
^^^^^^^^^^^^^^^^^^^
.. code-block:: yaml

   apiVersion: test.openstack.org/v1beta1
   kind: Tempest
   metadata:
     name: ci-gate-tests
     namespace: openstack
   spec:
     rerunFailedTests: true
     rerunOverrideStatus: true
     tempestRun:
       includeList: "tempest.api"
       concurrency: 4

Flaky tests that pass on the second attempt will not block the pipeline.
Tests with genuine failures will fail both times, and the pipeline
will correctly report a failure.

.. code-block:: yaml

   spec:
     rerunFailedTests: true
     rerunOverrideStatus: false

Used when collecting data on which tests are flaky without
changing your pipeline's pass/fail behavior. The pod still reports the
original failure, but the re-run logs show which tests recovered — useful
for building a skip list or reporting flakiness to upstream.

Using Re-run With Workflows
^^^^^^^^^^^^^^^^^^^^^^^^^^^^

When using workflows, the re-run setting can be configured differently for
each step. Each workflow step produces its own pod, and the re-run behavior
is independent per step:

.. code-block:: yaml

   apiVersion: test.openstack.org/v1beta1
   kind: Tempest
   metadata:
     name: full-validation
     namespace: openstack
   spec:
     rerunFailedTests: false
     tempestRun:
       includeList: "tempest.api"
     workflow:
       - stepName: api-validation
         rerunFailedTests: true
         rerunOverrideStatus: true
         tempestRun:
           includeList: "tempest.api.compute tempest.api.network"
       - stepName: scenario-tests
         rerunFailedTests: false
         tempestRun:
           includeList: "tempest.scenario"

In this example, API tests (which are more likely to be affected by transient
conditions) get the re-run safety net, while scenario tests (which are more
deterministic) fail immediately on first failure.

.. note::
   The per-step :code:`rerunFailedTests: true` overrides the base spec's
   :code:`rerunFailedTests: false` for that step only.

Checking Re-run Results
^^^^^^^^^^^^^^^^^^^^^^^^

The re-run results are stored in the same persistent volume as the original
run. To access them:

.. code-block:: bash

   oc logs <tempest-pod-name>

The container logs will show both the original run summary and the re-run
summary, making it clear which tests were retried and whether they passed.

.. _getting-logs:

Getting Logs
------------
The test-operator creates a persistent volume that is attached to a pod executing
the tests. Once the pod completes test execution, the pv contains all the artifacts
associated with the test run.

.. note::
   Please keep in mind that all resources created by the test operator are bound
   to the CR. Once you remove the CR (e.g. :code:`tempest/tempest-tests`), then
   you also remove the PV containing the logs.

If you want to retrieve the logs from the pv, you can follow these steps:

1. Spawn a pod with the pv attached to it.

.. code-block:: yaml

    ---
    apiVersion: v1
    kind: Pod
    metadata:
      name: test-operator-logs-pod
      namespace: "openstack"
    spec:
      containers:
        - name: test-operator-logs-container
          image: quay.io/quay/busybox
          command: ["/bin/sh", "-c", "--"]
          args: ["while true; do sleep 30; done;"]
          volumeMounts:
            - name: logs-volume
              mountPath: /mnt
      volumes:
        - name: logs-volume
          persistentVolumeClaim:
            # Note: In case you created your own custom resource then you
            #       have to put here the value from metadata.name.
            claimName: tempest-tests

2. Get an access to the logs by connecting to the pod created in the first
step:

.. code-block:: bash

   oc rsh pod/test-operator-logs-pod
   cd /mnt

**OR** get an access to the logs by copying the artifacts out of the pod created
in the first step:

.. code-block:: bash

   mkdir test-operator-artifacts
   oc cp test-operator-logs-pod:/mnt ./test-operator-artifacts
