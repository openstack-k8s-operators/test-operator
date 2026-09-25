Images Used by test-operator
============================

The `test-operator` can run images coming from two different builds. Both builds
publish to `quay.io <https://quay.io/>`_ and both can be used with the same custom
resources. The only thing that differs is the value of the :code:`containerImage`
parameter.

* **S2I images (default)** are built from the upstream source code on top of UBI
  in the
  `s2i-openstack-containers <https://github.com/openstack-k8s-operators/s2i-openstack-containers>`_
  repository and published to the
  `openstack-s2i-containers organization <https://quay.io/organization/openstack-s2i-containers>`_.
  The images tracking the `master` branch are tagged `master-latest`. These images
  are used when you do not set the :code:`containerImage` parameter.

* **TCIB images** are built from RPM packages in the
  `TCIB (The Container Image Build) <https://github.com/openstack-k8s-operators/tcib>`_
  repository and published to the
  `podified-antelope-centos9 organization <https://quay.io/organization/podified-antelope-centos9>`_
  with the `current-podified` tag. These images used to be the default. They are
  still built and still supported, set :code:`containerImage` in your CR if you
  want to keep using them.

.. note::
    Switching between the two builds does not require any other change in the CR.
    The test frameworks, the parameters and the produced artifacts stay the same.

.. _tempest-images:

Tempest Images
--------------

.. note::
    You can use the images to run a container or a pod on your own, without
    running the test-operator, see `Run Tempest in a Pod <./tempest_pod.html>`_
    or `Run Tempest via Podman <./tempest_podman.html>`_.

S2I tempest image
^^^^^^^^^^^^^^^^^

* `openstack-tempest <https://quay.io/openstack-s2i-containers/openstack-tempest>`__

  The default image. It installs tempest together with a set of tempest plugins,
  all of them built from the upstream source code. The list of the plugins that
  are installed has a tendency to change. Therefore for the up to date list check
  the `sources.txt <https://github.com/openstack-k8s-operators/s2i-openstack-containers/blob/main/containers/tempest/sources.txt>`_
  file of the image. There is only one S2I tempest image, plugins that are not
  part of it can still be installed during the container execution using the
  `tempestRun.externalPlugin*` parameters (see :ref:`tempest-custom-resource`).

TCIB tempest images
^^^^^^^^^^^^^^^^^^^

To find all TCIB tempest images, go to
`podified-master-centos9 organization <https://quay.io/organization/podified-master-centos9>`_
and filter for *tempest* results.

Currently, there are the following TCIB tempest images:

* `openstack-tempest <https://quay.io/podified-antelope-centos9/openstack-tempest>`__

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

Tobiko Image
------------

* `openstack-tobiko <https://quay.io/openstack-s2i-containers/openstack-tobiko:master-latest>`__

  The default image. It installs tobiko directly from the source code downloaded
  from `x/tobiko <https://opendev.org/x/tobiko.git>`_ repository.

* `openstack-tobiko <https://quay.io/podified-antelope-centos9/openstack-tobiko:current-podified>`__

  The TCIB variant of the same image.

AnsibleTest Image
-----------------

* `openstack-ansible-test <https://quay.io/openstack-s2i-containers/openstack-ansible-test:master-latest>`_

  The default image. It contains the ansible runtime and the collections that are
  needed to execute the playbooks referenced from the :code:`AnsibleTest` CR.

* `openstack-ansible-tests <https://quay.io/podified-antelope-centos9/openstack-ansible-tests:current-podified>`_

  The TCIB variant of the same image.

HorizonTest Image
-----------------

* `openstack-horizontest <https://quay.io/repository/openstack-s2i-containers/openstack-horizontest?tab=tags&tag=master-latest>`__

  The default image. It installs horizon directly from the source code downloaded
  from `x/horizon <https://opendev.org/openstack/horizon.git>`_ repository and
  prepares the environment for selenium tests to run.

* `openstack-horizontest <https://quay.io/repository/podified-antelope-centos9/openstack-horizontest?tab=tags&tag=current-podified>`__

  The TCIB variant of the same image.
