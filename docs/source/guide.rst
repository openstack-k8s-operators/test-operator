User Guide
==========

This guide is a starting point for configuring and running the `test-operator`.


Running Operator Locally Outside The Cluster
--------------------------------------------
This is **quick and easy way** how to experiment with the operator during development of a
new feature.

.. code-block:: bash

    make install run

Note, that after running the following command you will need to switch to
another terminal unless you run it in the background.

Running Operator Using The Operator Lifecycle Manager (OLM)
-----------------------------------------------------------

Another option is to start the operator by running the pre-build operator image
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

1. Create **OperatorGroup**

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

2. Create **CatalogSource**

.. code-block:: yaml

   cat catalog-source.yaml
   ---
   apiVersion: operators.coreos.com/v1alpha1
   kind: CatalogSource
   metadata:
     name: test-operator-index
     namespace: openstack
   spec:
     sourceType: grpc
     image: quay.io/openstack-k8s-operators/test-operator-index:latest

.. code-block:: bash

   oc apply -f catalog-source.yaml

3. Create **Subscription**

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
     source: test-operator-index
     sourceNamespace: openstack

.. code-block:: bash

   oc apply -f subscription.yaml

4. Wait for the **test-operator-controller-manager** pod to successfully spawn. Once you see
   the pod running you can start to communicate with the operator using the **Tempest** resource
   defined below.

.. code-block:: bash

   oc get pods
   ...
   test-operator-controller-manager-6c9994847c-6jwn5                 2/2     Running     0              20s
   ...


Executing Tempest Tests
-----------------------
.. _Executing Tempest Tests:

Once you have an operator running, then you can apply a tempest resource
definition, e.g. the default one:

.. literalinclude:: ../../config/samples/test_v1beta1_tempest.yaml
   :language: yaml

.. code-block:: bash

    oc apply -f config/samples/test_v1beta1_tempest.yaml

After that, verify that a pod was created with:

.. code-block:: bash

    oc get pods | grep tempest

You should see a pod with a name like `tempest-tests-xxxxx`.

To see the console output of the execution run the following:

.. code-block:: bash

    oc logs <name of the pod>

Custom Tempest Configuration
----------------------------
To configure tempest via tempest.conf use the `tempestconfRun.overrides`
parameter. This parameter accepts a list of key value pairs that specify values
that should be written to tempest.conf generated inside the container.

For example this definition of Tempest object:

.. code-block:: yaml

    ---
    apiVersion: test.openstack.org/v1beta1
    kind: Tempest
    metadata:
      name: tempest-tests
      namespace: openstack
    spec:
      containerImage: quay.io/podified-antelope-centos9/openstack-tempest:current-podified
      tempestRun:
        includeList: | # <-- Use | to preserve \n
          tempest.api.identity.v3.*
        concurrency: 8
      tempestconfRun:
          overrides: |
            auth.admin_username admin
            auth.admin_password 1234

will ensure that tempest will be executed with tempest.conf that looks like this:


.. code-block:: toml

   ...
   [auth]
   admin_username = admin
   admin_password = 1234
   ...


Getting Logs
------------
This is TBA.
