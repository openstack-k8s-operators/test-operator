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
* `openstack-tempest-extras <https://quay.io/podified-antelope-centos9/openstack-tempest-extras>`_

`test-operator` runs, for now, only the following test images:

* openstack-tempest
