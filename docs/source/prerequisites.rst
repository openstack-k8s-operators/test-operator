.. _prerequisites:

Prerequisites
=============

First, you will need an OpenStack Platform running on OpenShift. See, the
`ci-framework documentation <https://ci-framework.readthedocs.io/en/latest/>`_
to get you started. It will get you through the installation of such environment.

E.g. if you want to install OpenStack Platform running on OpenShift on your
own hardware, follow `the steps in this doc <https://ci-framework.readthedocs.io/en/latest/quickstart/04_non-virt.html>`_.

After the installations is completed, you can source the credentials to the
environment as follows:

.. code-block:: bash

    eval $(${HOME}/ci-framework-data/bin/crc oc-env)
    export KUBECONFIG="${HOME}/.crc/machines/crc/kubeconfig"
    oc login -u kubeadmin -p 12345678 https://api.crc.testing:6443

.. note::
    12345678 is a default password set by ci-framework
