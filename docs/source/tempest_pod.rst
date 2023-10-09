Run Tempest in a Pod
====================
Sometimes, you may want to avoid running the whole operator, e.g. for
debugging or development purposes. In that case, you can run Tempest in a
`pod <https://docs.openshift.com/online/pro/architecture/core_concepts/pods_and_services.html>`_.

Write Object Definitions
--------------------------

Create a file named ``tempest-config.yaml`` with the following content:

.. literalinclude:: samples/tempest-config.yaml
   :language: yaml

The file contains tempest configuration and will be used to pass any tempest
options to the container running Tempest.

Then create a file named ``tempest-deployment.yaml`` with the following content:

.. literalinclude:: samples/tempest-deployment.yaml
   :language: yaml

The file contains a pod definition - it tells OpenShift to create a pod running
one container that will run Tempest from the openstack-tempest image. More about
images can be found `here <./images.html>`_.

Create Resources
----------------

Run the `oc apply` command to create the resources:

.. code-block:: bash

    oc apply -f tempest-config.yaml
    oc apply -f tempest-deployment.yaml

You can verify that the resources were created with:

.. code-block:: bash

    oc get configmap my-tempest-data
    oc get pods tempest-worker

Add ``-o yaml`` to the above commands to see the details of the resources.

Debug Tempest Container
-----------------------

Use ``oc describe`` command to see the full definition of the pod including
latest events, such as pulling of the tempest image, creating and starting
a container.

.. code-block:: bash

    oc describe pod tempest-worker

To see the console output from the tempest run execute:

.. code-block:: bash

    oc logs tempest-worker

SSH to the Tempest Container
````````````````````````````
In case you want to SSH the container to run Tempest manually, you may want to
run the pod as a `sleepy` pod. Uncomment the ``command`` option in the
``tempest-deployment.yaml`` file shared above to create such pod.

Then SSH to the container:

.. code-block:: bash

    oc rsh --container tempest-container tempest-worker

.. note::
    If a pod runs only one container you can omit ``--container`` argument.

Once inside the container, change directory to ``/var/lib/tempest`` to find
tempest configuration together with the ``run_tempest.sh`` script.

While inside the container, you can run Tempest commands according to the
`Tempest documentation <https://docs.openstack.org/tempest/latest/>`_.

The container has also installed
`python-tempestconf <https://opendev.org/openinfra/python-tempestconf>`_ project
(``discover-tempest-config`` command). If you run the pod as a `sleepy` one,
``run_tempest.sh`` defined in the image wasn't executed. In that case, you have
to generate ``tempest.conf`` manually - either run ``run_tempest.sh`` or
``discover-tempest-config`` command according to
`its documentation <https://docs.opendev.org/openinfra/python-tempestconf/latest/>`_.

Custom Tempest Configuration
----------------------------
In order to pass custom configuration to `tempest` and `python-tempestconf`, you
can either SSH to the running tempest container where you can interact with
``tempest`` and ``discover-tempest-config`` commands directly according to their
documentations, see `SSH to the Tempest Container`_, or follow the steps
described below.

python-tempestconf configuration
`````````````````````````````````
The only, currently, supported way for passing custom arguments to
``discover-tempest-config`` command is via a file called ``profile.yaml``. See
the python-tempestconf's official documentation for
`more details about the file <https://docs.opendev.org/openinfra/python-tempestconf/latest/user/profile.html>`_.

Add the content of the ``profile.yaml`` file to the ``tempest-config.yaml`` file
under **data** section like this:

.. code-block:: yaml

    data:
      <any existing configuration>
      profile.yaml: |
        <file content>

And then add the following to the ``tempest-deployment.yaml`` under
**volumeMounts** section of the tempest-container:

.. code-block:: yaml

    - mountPath: "/var/lib/tempest/external_files/profile.yaml"
      name: tempest-config
      subPath: profile.yaml

Tempest configuration
`````````````````````
Change the ``tempest-config.yaml`` file accordingly to pass any custom
configuration to Tempest. Please mind the content of the ``run_tempest.sh``
script defined inside the tempest image as that is the limitation of what
can be recognized, parsed and taken into account during the tempest run.

The content of the ``run_tempest.sh`` `can be found here <https://github.com/openstack-k8s-operators/tcib/blob/main/container-images/tcib/base/os/tempest/run_tempest.sh>`_.
