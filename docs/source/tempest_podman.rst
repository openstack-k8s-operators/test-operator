Run Tempest via Podman
======================
Sometimes, you may want to avoid running the whole operator, e.g. for
debugging or development purposes. In that case, you can run Tempest locally
via Podman.

This is probably the fastest way to run tempest, but it's the least
user-friendly one.

Create a tempest directory where you will put your config files and where
tempest will save the logs from the test run:

.. code-block:: bash

    mkdir -p /tmp/tempest/logs

Find/generate clouds.yaml file. `ci-framework` has a Jinja template for
generating the file, `see <https://github.com/openstack-k8s-operators/ci-framework/blob/a30b3d7d958f9d3bf9091178929352993573a4b0/ci_framework/roles/tempest/tasks/configure-tempest.yml#L28-L31>`_.
Once you have ``clouds.yaml`` copy it to ``/tmp/tempest/logs``:

.. code-block:: bash

    cp ~/clouds.yaml /tmp/tempest/logs

Create ``exclude.txt`` and ``include.txt`` files

.. code-block:: bash

    touch /tmp/tempest/logs/exclude.txt
    touch /tmp/tempest/logs/include.txt

Include the tempest test(s) you want to run in ``include.txt`` file.

Set directory permissions:

.. code-block:: bash

    podman unshare chown 42480:42480 -R /tmp/tempest/logs

Run tempest:

.. code-block:: bash

    podman run -e CONCURRENCY=4 -v /tmp/tempest/logs/:/var/lib/tempest/external_files:Z quay.io/podified-antelope-centos9/openstack-tempest:current-podified

Profit! Logs will be in ``/tmp/tempest/logs`` (subunit, html files, tempest logs, etc)
