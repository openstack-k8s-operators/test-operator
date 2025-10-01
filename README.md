# test-operator
This is a test operator that automates testing of an OpenStack deployment
running on an OpenShift cluster. The operator supports execution of the following
tests:

- [Tempest](https://opendev.org/openstack/tempest)

- [Tobiko](https://tobiko.readthedocs.io/en/master)

- [Horizon tests](https://opendev.org/openstack/horizon)

- Arbitrary tests written as an Ansible role (refer to **AnsibleTest CR**)


## Getting Started
First, you will need an OpenStack Platform running on top of OpenShift. See:

- [ci-framework documentation](https://ci-framework.readthedocs.io/en/latest/) or

- [install_yamls GitHub repository](https://github.com/openstack-k8s-operators/install_yamls/blob/main/README.md)

to get you started. It will get you through the installation of such environment.

Then proceed to our [documentation](https://openstack-k8s-operators.github.io/test-operator/).

## Contributing
Please, make sure that pre-commit checks pass prior to proposing a PR.

1. Install pre-commit tool

```bash
python3 -m venv .venv
. .venv/bin/activate
python3 -m pip install pre-commit
pre-commit install --install-hooks
```

2. Execute pre-commit checks

```bash
pre-commit run --show-diff-on-failure --color=always --all-files --verbose
```

### Test It Out
Run the following commands if you want to quickly test your test-operator related
changes locally.

1. Install the CRDs into the cluster:

```sh
make install
```

2. Run your controller (this will run in the foreground, so switch to a new
   terminal if you want to leave it running):

```sh
ENABLE_WEBHOOKS=false make run
```

**NOTE:** You can also run this in one step by running: `ENABLE_WEBHOOKS=false make install run`

Now that test-operator is automatically deployed in the podified environment,
you need to additionally delete the test-operator controller manager to test
local changes. For more information, check out the section *Running Test Operator Locally*
in our documentation.

## License

Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
