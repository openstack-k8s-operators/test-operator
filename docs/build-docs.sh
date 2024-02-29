#!/usr/bin/env bash

set -euxo pipefail

TEMP_VENV_ENV="$(mktemp -d)"
DOCS_DIR="./docs"

python -m venv ${TEMP_VENV_ENV} && source ${TEMP_VENV_ENV}/bin/activate

pip install -c ${UPPER_CONSTRAINTS_FILE:-https://releases.openstack.org/constraints/upper/master} -r ${DOCS_DIR}/requirements.txt

# Run linter on docs, skipping antsibull-docs output as it isn't up to spec
doc8 --config ${DOCS_DIR}/doc8.ini ${DOCS_DIR}/source

sphinx-build -a -E -W -d ${DOCS_DIR}/build/doctrees --keep-going -b html ${DOCS_DIR}/source ${DOCS_DIR}/build/html -T

deactivate
