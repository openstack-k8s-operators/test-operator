FAQ
===

**1. There is a test failing and I do not know why. How can I debug the failing
test?**

Currently, there are two ways how to debug a test failure. You can either use
:code:`oc rsh` or :code:`oc debug` command. The preferred way is to use the
:code:`oc debug` command as it does not terminate the connection to the test pod
once the test pod finishes its execution.

.. note::

   **What does** :code:`oc debug` **do?**

   $ oc debug --help

   Launch a command shell to debug a running application.

   When debugging images and setup problems, it's useful to get an exact copy
   of a running pod configuration and troubleshoot with a shell. Since a pod
   that is failing may not be started and not accessible to 'rsh' or 'exec', the
   'debug' command makes it easy to create a **carbon copy of that setup**.

Once connected to a test pod using :code:`oc debug`, you need to execute the
script responsible for the test execution (e.g., in the case of tempest
`/var/lib/tempest/run_tempest.sh`) as the :code:`oc debug` prepares a pod with
an environment that is identical to that of a freshly started pod (meaning the
environment looks exactly how it would look like **before** the tests are executed
and configured).

After the script finishes due to a failure you can start debugging the issue
(e.g., by using `pudb <https://pypi.org/project/pudb/>`_ or any other debugger
of your choosing).


**2. When I apply the test-operator related CR (e.g.,** :code:`Tempest` **CR) I get**
:code:`resource mapping not found` **error. What's wrong?**

.. code-block::

    $ oc apply -f file.yml
    error: resource mapping not found for name: "tempest" namespace: "openstack" from "file.yml": no matches for kind "Tempest" in version
    "test.openstack.org/v1beta1"

This issue in most cases is related to the test-operator not being (properly)
installed. Try finding an answer to the following questions:

- Do you have the test-operator related CRDs installed
  (:code:`oc get crd | grep -E "tempest|tobiko"`)?

- Do you have the controller running in your environment
  (:code:`oc get pod | grep "test-operator-controller"`)?

If you answered **NO** to any of those two questions, then the best solution
is to uninstall and install the test-operator again. Please, refer to these
two sections from the test-operator documentation:

- :ref:`uninstalling-operator`

- :ref:`running-operator-olm`


**3. I have a patch for tempest or a tempest plugin that I want to test using
the test-operator. How can I make sure that the patch is used by the
test-operator?**

The :code:`Tempest` CR contains a parameter called :code:`externalPlugin` that
can be used to specify a tempest / tempest plugin patch that should be used inside
a test pods spawned by the test operator. The parameter requires three pieces
of information (:code:`repository`, :code:`changeRepository` and
:code:`changeRefspec`).

Let's say I want to download a patch proposed via gerrit to neutron-tempest-plugin.
In this case I would need to add :code:`externalPlugin` section under the
:code:`tempestRun` section in the :code:`Tempest` CR:

.. code-block:: yaml

    externalPlugin:
        - repository: https://opendev.org/openstack/neutron-tempest-plugin.git"
          changeRepository: "https://review.opendev.org/openstack/neutron-tempest-plugin"
          changeRefspec: "refs/changes/97/896397/2"


In the example above, the change specified with :code:`changeRefspec` will be
applied inside the test pod to code stored in the :code:`/var/lib/tempest/external-plugins`
folder.

.. note::

    This change is only available for the :code:`Tempest` CR.


**4. I need to pass a private / public key to a test pod via parameter in the
test-operator related CR (e.g.,** :code:`SSHKeySecretName` **in the**
:code:`Tempest` **CR), but the CR requires a name of a secret containing the key.
How can I create a secret that stores the key?**

To create a secret that contains private / public key, the :code:`oc secret create`
command can be utilized:

.. code-block:: bash

    $ oc create secret generic my-private-key-secret --from-file=ssh-privatekey=/path/to/private/key/file


**5. I want to execute tests from tempest plugin XYZ, but the tests seem to be
missing. What should I do?**

Please refer to this section of the documentation:

- :ref:`tempest-images`

Most likely you are using openstack-tempest image instead of openstack-tempest-all.
You can specify which image you want to be used with :code:`containerImage`
parameter in the :code:`Tempest` CR. If your plugin is not included in the
:code:`openstack-tempest-all` image then take a look at the :code:`externalPlugin`
parameter.


**6. The test pod is stuck in a pending state. What should I do?**

There are a lot of things that might lead to a pod being stuck in a Pending
state. Usually, the best approach is to investigate what went wrong using the
:code:`oc describe pod/[pod-name]` command (see :code:`Event` section).

.. warning::

    Make absolutely sure that you have a working backup of the data residing on
    that PV (if needed) before wiping the data.

However, most of the times this issue is caused by the fact that there are no
available PVs left on your system. This happens when you executed tests
too many times and test-operator is told to use storage class (using the
:code:`storageClass` option) which does not clean up the PVs after itself.

The issue can be fixed in two steps:

1. Identify test-operator related PVs on your system
   (e.g., by running :code:`oc get pv | grep "tempest"`)

2. Modify the PVs identified in the first step using the :code:`oc edit` command
   so that value under :code:`ClaimRef` is changed to :code:`null`. This will
   free the PVs and you can continue with the testing but please make sure that
   you can afford to lose the data stored on the PVs you are about the free.


**7. How can I build test-operator-index locally?**

To build test-operator locally you can follow these steps inside `/test-operator`
directory. Be sure to double-check which image name is required for each step.

1. Create test-operator image

.. code-block:: bash

    $ make docker-build IMG=<registry>/<user>/<operator_image_name>:<tag>
    $ make docker-push IMG=<registry>/<user>/<operator_image_name>:<tag>

2. Create test-operator-bundle image

.. code-block:: bash

    $ make bundle IMG=<registry>/<user>/<operator_image_name>:<tag>
    $ make bundle-build BUNDLE_IMG=<registry>/<user>/<bundle_image_name>:<tag>
    $ make bundle-push BUNDLE_IMG=<registry>/<user>/<bundle_image_name>:<tag>

3. Create test-operator-index image

.. code-block:: bash

    $ make catalog-build CATALOG_IMG=<registry>/<user>/<index_image_name>:<tag>
    $ make catalog-push CATALOG_IMG=<registry>/<user>/<index_image_name>:<tag>
