Images Used by test-operator
============================

The test-operator uses images from two different builds, both published to
quay.io. **S2I images** (default) are built from upstream source code in
`s2i-openstack-containers <https://github.com/openstack-k8s-operators/s2i-openstack-containers>`_
and published to
`openstack-s2i-containers <https://quay.io/organization/openstack-s2i-containers>`_ organization
with `master-latest` tag. **TCIB images** are built from RPM packages in
`TCIB <https://github.com/openstack-k8s-operators/tcib>`_ and published to
`podified-antelope-centos9 <https://quay.io/organization/podified-antelope-centos9>`_ organization
with `current-podified` tag.

.. _tempest-images:

Tempest Images
--------------

.. note::
    You can use the images to run a container or a pod on your own, without
    running the test-operator, see `Run Tempest in a Pod <./tempest_pod.html>`_
    or `Run Tempest via Podman <./tempest_podman.html>`_.

S2I tempest image
^^^^^^^^^^^^^^^^^

* `openstack-tempest <https://quay.io/openstack-s2i-containers/openstack-tempest>`__ (default)

  Installs tempest together with a set of tempest plugins, all built from upstream
  source code. For the up to date list check
  `sources.txt <https://github.com/openstack-k8s-operators/s2i-openstack-containers/blob/main/containers/tempest/sources.txt>`_.
  Plugins not included can be installed using :code:`externalPlugin` parameter.

TCIB tempest images
^^^^^^^^^^^^^^^^^^^

To find all TCIB tempest images, go to
`podified-master-centos9 organization <https://quay.io/organization/podified-master-centos9>`_
and filter for *tempest* results.

Currently, there are the following TCIB tempest images:

* `openstack-tempest <https://quay.io/podified-antelope-centos9/openstack-tempest>`__

  An image that installs `openstack-tempest` RPM package that contains only tempest and no other
  plugins. The user can install any external plugin during the container execution using
  the :code:`externalPlugin` parameter.

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

Tobiko Image
------------

* `openstack-tobiko <https://quay.io/openstack-s2i-containers/openstack-tobiko:master-latest>`__ (S2I - default)

  Installs tobiko from source code downloaded from
  `x/tobiko <https://opendev.org/x/tobiko.git>`_ repository.

* `openstack-tobiko <https://quay.io/podified-antelope-centos9/openstack-tobiko:current-podified>`__ (TCIB)

AnsibleTest Image
-----------------

* `openstack-ansible-test <https://quay.io/openstack-s2i-containers/openstack-ansible-test:master-latest>`__ (S2I - default)

  Contains ansible runtime and collections needed to execute playbooks referenced
  from the :code:`AnsibleTest` CR.

* `openstack-ansible-tests <https://quay.io/podified-antelope-centos9/openstack-ansible-tests:current-podified>`__ (TCIB)

HorizonTest Image
-----------------

* `openstack-horizontest <https://quay.io/repository/openstack-s2i-containers/openstack-horizontest?tab=tags&tag=master-latest>`__ (S2I - default)

  Installs horizon from source code downloaded from
  `x/horizon <https://opendev.org/openstack/horizon.git>`_ repository and
  prepares environment for selenium tests.

* `openstack-horizontest <https://quay.io/repository/podified-antelope-centos9/openstack-horizontest?tab=tags&tag=current-podified>`__ (TCIB)
