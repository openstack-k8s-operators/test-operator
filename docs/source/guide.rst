Run Tests via test-operator
===========================

.. note::
   Before you proceed with this section of the documentation please make sure
   that you read the :ref:`prerequisites <prerequisites>`.

This guide describes:

* How to **install/uninstall** the operator?

  * :ref:`running-operator-olm`

  * :ref:`running-operator-locally`

  * :ref:`uninstalling-operator`

* How to **run tests** via the the operator?

  * :ref:`executing-tests`

  * :ref:`getting-logs`

If you want to get your hands on the test-operator quickly then follow these two
sections: :ref:`running-operator-olm` and :ref:`executing-tests`.

.. _running-operator-olm:

Running Operator Using the Operator Lifecycle Manager (OLM)
-----------------------------------------------------------

The first option of how to start the operator is by running the pre-build operator image
stored on
`quay.io <https://quay.io/repository/openstack-k8s-operators/test-operator>`_
using the OLM.

.. note::

   Currently, the `test-operator <https://quay.io/openstack-k8s-operators/test-operator>`_ is not
   part of the `openstack-operator-index <https://quay.io/openstack-k8s-operators/
   openstack-operator-index>`_ therefore a new catalog source which uses `test-operator-index
   <https://quay.io/openstack-k8s-operators /test-operator-index>`_ image needs to be created
   in advance.

Follow these steps to install the operator in the openstack project.

1. Create :code:`OperatorGroup`

.. code-block:: yaml

   cat operator-group.yaml
   ---
   apiVersion: operators.coreos.com/v1
   kind: OperatorGroup
   metadata:
     name: openstack-operatorgroup
     namespace: openstack
   spec:
     targetNamespaces:
       - openstack

.. code-block:: bash

   oc apply -f operator-group.yaml

2. Create :code:`CatalogSource`

.. code-block:: yaml

   cat catalog-source.yaml
   ---
   apiVersion: operators.coreos.com/v1alpha1
   kind: CatalogSource
   metadata:
     name: test-operator-catalog
     namespace: openstack
   spec:
     sourceType: grpc
     image: quay.io/openstack-k8s-operators/test-operator-index:latest

.. code-block:: bash

   oc apply -f catalog-source.yaml

3. Create :code:`Subscription`

.. code-block:: yaml

   cat subscription.yaml
   ---
   apiVersion: operators.coreos.com/v1alpha1
   kind: Subscription
   metadata:
     name: test-operator
     namespace: openstack
   spec:
     name: test-operator
     source: test-operator-catalog
     sourceNamespace: openstack

.. code-block:: bash

   oc apply -f subscription.yaml

4. Wait for the :code:`test-operator-controller-manager` pod to successfully
   spawn. Once you see  the pod running you can start to communicate with the
   operator using the :code:`Tempest` resource defined in the
   :ref:`executing-tests` section.

.. code-block:: bash

   oc get pods
   ...
   test-operator-controller-manager-6c9994847c-6jwn5                 2/2     Running     0              20s
   ...


.. _running-operator-locally:

Running Operator Locally Outside the Cluster
--------------------------------------------
This is **quick and easy way** how to experiment with the operator during
development of a new feature.

.. code-block:: bash

    make install run

Note, that after running the following command you will need to switch to
another terminal unless you run it in the background.

.. _uninstalling-operator:

Uninstalling Operator
---------------------

If you installed the operator by following the steps in the
:ref:`running-operator-olm` section then this section can come handy. You
might need to uninstall the operator when:

* you encountered issues during the installation process or when

* you want to be sure that you are ussing the latest version of the operator.

Please, make sure that you follow the order of the steps:

1. Remove all instances of the :code:`Tempest` and :code:`Tobiko` CRDs

.. code-block:: bash

   oc get tempest

   NAME            AGE
   tempest-tests   3s


.. code-block:: bash

   oc delete tempest/tempest-tests
   oc delete tobiko/tobiko-tests

2. Remove the :code:`crd`

.. code-block:: bash

   oc delete crd/tempests.test.openstack.org
   oc delete crd/tobikoes.test.openstack.org

3. Remove the :code:`subscription` you created during
   :ref:`the installation <running-operator-olm>`.

.. code-block:: bash

   oc delete subscription/test-operator

4. Remove the :code:`catalog` source you created during
   :ref:`the installation <running-operator-olm>`.

.. code-block:: bash

   oc delete catalogsource/test-operator-catalog

6. Remove the :code:`operatorgroup` you created during
   :ref:`the installation <running-operator-olm>`.

.. code-block:: bash

   oc delete operatorgroup/openstack-operatorgroup

7. Remove the :code:`csv`

.. code-block:: bash

   oc delete csv/test-operator.v0.0.1

8. Remove the :code:`operator`. It is possible that if you executed
   the previous commands too quickly then you will need to execute this
   command twice.

.. code-block:: bash

   oc delete operator/test-operator.openstack

9. Check that there are no test-operator related resources hanging. This step
   is not required.

.. code-block:: bash

   oc get olm

.. note::
   It might happen that by changing the order of the uninstallation steps,
   you encounter a situation when you will not be able to delete the
   :code:`crd`. In such a case, try to delete the :code:`finalizers:`
   section in the output of the :code:`oc edit tempest/tempest-tests`.


.. _executing-tests:

Executing Tests
---------------

Once you have an operator running, then you can apply a custom resource accepted
by the test-operator to start the testing. Currently, two types of custom
resources are being accepted by the test-operator (see
:ref:`custom-resources-used-by-the-test-operator` section):

* :ref:`tempest-custom-resource`

* :ref:`tobiko-custom-resource`

1. Create a manifest for custom resource accepted by the test-operator
   (:ref:`custom-resources-used-by-the-test-operator` section).

2. Apply the manifest. Either go with the default one, the command below, or
   replace the path with a manifest created in the first step.

.. code-block:: bash

    oc apply -f config/samples/test_v1beta1_tempest.yaml

3. Verify that the pod executing the tests is running. It might take a couple
   of seconds for the test pod to spawn. Also, note that by default the test-operator
   allows only one test pod to be running at the same time (read
   :ref:`parallel-execution`). If you defined your own custom resource in the first step
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

2 (a). Get an access to the logs by connecting to the pod created in the fist
step:

.. code-block:: bash

   oc rsh pod/test-operator-logs-pod
   cd /mnt

2 (b). Or get an access to the logs by copying the artifacts out of the pod created
in the first step:

.. code-block:: bash

   mkdir test-operator-artifacts
   oc cp test-operator-logs-pod:/mnt ./test-operator-artifacts
