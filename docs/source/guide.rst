Run Tempest via test-operator
=============================

.. note::
   Before you proceed with this section of the documentation please make sure
   that you read the :ref:`prerequisites <prerequisites>`.

This guide describes:

* How to **install/uninstall** the operator?

  * Running Operator Locally Outside The Cluster

  * Running Operator Using the Operator Lifecycle Manager (OLM)

  * Uninstalling Operator

* How to **run tests** via the the operator?

  * Executing Tempest Tests

  * Custom Tempest Configuration

  * Getting Logs

If you want to get your hands on the test-operator quickly then follow these two
sections: :ref:`running-operator-olm` and :ref:`executing-tempest-tests`.

Running Operator Locally Outside the Cluster
--------------------------------------------
This is **quick and easy way** how to experiment with the operator during development of a
new feature.

.. code-block:: bash

    make install run

Note, that after running the following command you will need to switch to
another terminal unless you run it in the background.

.. _running-operator-olm:

Running Operator Using the Operator Lifecycle Manager (OLM)
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


Uninstalling Operator
---------------------

If you installed the operator by following the steps in the
:ref:`running-operator-olm` section then this section might come handy. You
might need to uninstall the operator when:

* you encountered issues during the installation process or when

* you want to be sure that you are ussing the latest version of the operator

Please, make sure that you follow the order of the steps:

1. Remove all instances of the **Tempest** CRD

.. code-block:: bash

   oc get tempest

   NAME            AGE
   tempest-tests   3s


.. code-block:: bash

   oc delete tempest/tempest-tests

2. Remove the CRD

.. code-block:: bash

   oc delete crd/tempests.test.openstack.org

3. Remove the subscription you created during
   :ref:`the installation <running-operator-olm>`.

.. code-block:: bash

   oc delete subscription/test-operator

4. Remove the catalog source you created during
   :ref:`the installation <running-operator-olm>`.

.. code-block:: bash

   oc delete catalogsource/test-operator-catalog

6. Remove the operator group you created during
   :ref:`the installation <running-operator-olm>`.

.. code-block:: bash

   oc delete operatorgroup/openstack-operatorgroup

7. Remove the csv

.. code-block:: bash

   oc delete csv/test-operator.v0.0.1

8. Remove the operator. It is possible that if you executed the previous commands
   too quickly then you might need to execute this command twice.

.. code-block:: bash

   oc delete operator/test-operator.openstack

9. Check that there are no test-operator related resources hanging. This step
   is not required.

.. code-block:: bash

   oc get olm

.. note::
   It might happend that by changing the order of the uninstallation steps
   you encounter a situation when you will not be able to delete the
   CRD. In such a case it might help to delete the :code:`finalizers:`
   section in the output of the :code:`oc edit tempest/tempest-tests`


.. _executing-tempest-tests:

Executing Tempest Tests
-----------------------

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
