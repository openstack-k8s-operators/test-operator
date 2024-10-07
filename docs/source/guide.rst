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
This is a **quick and easy way** to experiment with the operator during
development of a new feature.

.. code-block:: bash

    ENABLE_WEBHOOKS=false make install run

Note that after running the following command, you will need to switch to
another terminal unless you run it in the background.

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


.. _getting-logs:

Getting Logs
------------
The test-operator creates a persistent volume that is attached to a pod executing
the tests. Once the pod completes test execution, the pv contains all the artifacts
associated with the test run.

.. note::
   Please keep in mind that all resources created by the test operator are bound
   to the CR. Once you remove the CR (e.g.::code:`tempest/tempest-tests`), then
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
