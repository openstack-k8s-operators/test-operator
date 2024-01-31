.. _custom-resources-used-by-the-test-operator:

=============================
CRs Used By the test-operator
=============================

This file contains definitions of all custom resources (CRs) that are accepted by
the test-operator. Currently, there are two CRs accepted by the test-operator:

* :ref:`tempest-custom-resource`

* :ref:`tobiko-custom-resource`


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

