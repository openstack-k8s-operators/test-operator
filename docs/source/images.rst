Images Used by test-operator
============================

The `test-operator` uses images that are built and defined in
`TCIB (The Container Image Build) <https://github.com/openstack-k8s-operators/tcib>`_.
The built images are published to `quay.io <https://quay.io/>`_.

Tempest Images
--------------

.. note::
    You can use the images to run a container or a pod on your own, without
    running the test-operator, see `Run Tempest in a Pod <./tempest_pod.html>`_
    or `Run Tempest via Podman <./tempest_podman.html>`_.

To find all tempest images, go to
`podified-master-centos9 organization <https://quay.io/organization/podified-master-centos9>`_
and filter for *tempest* results.

Currently, there are the following tempest images:

* `openstack-tempest <https://quay.io/podified-antelope-centos9/openstack-tempest>`_

  An image that installs `openstack-tempest` RPM package that contains only tempest and no other
  plugins. The user can install any external plugin during the container execution using
  the `tempestRun.externalPlugin*` parameters (see :ref:`tempest-custom-resource`)

* `openstack-tempest-all <https://quay.io/podified-antelope-centos9/openstack-tempest-all>`_

  An image that installs `openstack-tempest-all` RPM package. Most of the tempest plugins are
  included in the RPM too, see `the spec file <https://github.com/rdo-packages/tempest-distgit/blob/rpm-master/openstack-tempest.spec>`_
  for the exact list.

* `openstack-tempest-extras <https://quay.io/podified-antelope-centos9/openstack-tempest-extras>`_

  An image that installs `openstack-tempest-all` RPM package. On top of the all the plugins that are part of the RPM,
  this image contains a few extras. The list of the extra projects (mainly tempest plugins) that are installed there has
  a tendency to change. Therefore for the up to date list check the
  `TCIB definition <https://github.com/openstack-k8s-operators/tcib/blob/main/container-images/tcib/base/os/tempest/tempest-extras/tempest-extras.yaml>`_
  of the image.


`test-operator` runs, for now, only the following test images:

* `openstack-tempest <https://quay.io/podified-antelope-centos9/openstack-tempest>`_

Tobiko Image
------------

* `openstack-tobiko <https://quay.io/podified-antelope-centos9/openstack-tobiko:current-podified>`_

  An image that installs tobiko directly from the source code downloaded from
  `x/tobiko <https://opendev.org/x/tobiko.git>`_ repository.
