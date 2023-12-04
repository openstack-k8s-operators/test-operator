Test Images
===========

The `test-operator` uses images that are built and defined in
`TCIB (The Container Image Build) <https://github.com/openstack-k8s-operators/tcib>`_.
The built images are published to `quay.io <https://quay.io/>`_.

.. note::
    You can use the images to run a container or a pod on your own, without
    running the test-operator, see `Run Tempest in a Pod <./tempest_pod.html>`_
    or `Run Tempest via Podman <./tempest_podman.html>`_.

To find all tempest images, go to
`podified-master-centos9 organization <https://quay.io/organization/podified-master-centos9>`_
and filter for *tempest* results.

Currently, there are the following tempest images:

* `openstack-tempest <https://quay.io/podified-antelope-centos9/openstack-tempest>`_

  An image that contains only tempest and no other plugins. The user can install any external
  plugin during the container execution using the `tempestRun.externalPlugin*` parameters
  (see :ref:`Executing Tempest Tests<Executing Tempest Tests>`)

* `openstack-tempest-all <https://quay.io/podified-antelope-centos9/openstack-tempest-all>`_

  An image that contains tempest and all plugins that have an rpm:

  `neutron-tests-tempest, networking-l2gw, trove-tempest-plugin, ironic-tempest-plugin,
  cinder-tempest-plugin, manila-tempest-plugin, designate-tempest-plugin, octavia-tempest-plugin,
  barbican-tempest-plugin, keystone-tempest-plugin, novajoin-tempest-plugin,
  kuryr-tempest-plugin, magnum-tempest-plugin, mistral-tempest-plugin, murano-tempest-plugin,
  patrole, watcher-tempest-plugin, zaqar-tempest-plugin, heat-tempest-plugin,
  telemetry-tempest-plugin, sahara-tempest-plugin, sahara-tests, vitrage-tempest-plugin.`

* `openstack-tempest-extras <https://quay.io/podified-antelope-centos9/openstack-tempest-extras>`_

  An image that contains `tempest-stress` and `whitebox-tempest-plugin` on top of the all plugins
  that are part of the `openstack-tempest-all` image.


`test-operator` runs, for now, only the following test images:

* `openstack-tempest <https://quay.io/podified-antelope-centos9/openstack-tempest>`_
