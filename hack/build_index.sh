#!/bin/bash
#
# This command builds test-operator index and pushes it to quay.io. Podman
# should be authenticated against quay.io prior to the execution of this
# command.
#
#   USER=quay-nickname TAG=xyz ./hack/build_index.sh
#

set -x

USER=${USER:-$USERNAME}
REGISTRY=${REGISTRY:-quay.io}
IMAGE_NAME=${IMAGE_NAME:-test-operator}
BUNDLE_IMAGE_NAME=${BUNDLE_IMAGE_NAME:-test-operator-bundle}
INDEX_IMAGE_NAME=${INDEX_IMAGE_NAME:-test-operator-index}
TAG=${TAG:-latest}

make docker-build IMG=${REGISTRY}/${USER}/${IMAGE_NAME}:${TAG}
make docker-push IMG=${REGISTRY}/${USER}/${IMAGE_NAME}:${TAG}

make bundle IMG=${REGISTRY}/${USER}/${IMAGE_NAME}:${TAG}
make bundle-build BUNDLE_IMG=${REGISTRY}/${USER}/${BUNDLE_IMAGE_NAME}:${TAG}
make bundle-push BUNDLE_IMG=${REGISTRY}/${USER}/${BUNDLE_IMAGE_NAME}:${TAG}

export BUNDLE_IMG=${REGISTRY}/${USER}/${BUNDLE_IMAGE_NAME}:${TAG}
make catalog-build CATALOG_IMG=${REGISTRY}/${USER}/${INDEX_IMAGE_NAME}:${TAG}
make catalog-push CATALOG_IMG=${REGISTRY}/${USER}/${INDEX_IMAGE_NAME}:${TAG}
