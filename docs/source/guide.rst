User Guide
==========

This guide is a starting point for configuring and running the `test-operator`.

Installation
------------

.. code-block:: bash

    make install

Execution
---------
Execute the following command to run the operator. Note, that after running the
following command you will need to switch to another terminal unless you run it
in the background.

.. code-block:: bash

    make run

.. note::
    You can run this step together with the installation at once by running: ``make install run``

Then apply a tempest resource definition, e.g. the default one:

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
This is TBA.

Getting logs
------------
This is TBA.
