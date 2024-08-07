#!/bin/bash
set -ex

TMPDIR=${TMPDIR:-"/tmp/k8s-webhook-server/serving-certs"}
SKIP_CERT=${SKIP_CERT:-false}
CRC_IP=${CRC_IP:-$(/sbin/ip -o -4 addr list crc | awk '{print $4}' | cut -d/ -f1)}
FIREWALL_ZONE=${FIREWALL_ZONE:-"libvirt"}
SKIP_FIREWALL=${SKIP_FIREWALL:-true}

if [ "$SKIP_FIREWALL" = false ] ; then
    #Open 9443
    sudo firewall-cmd --zone=${FIREWALL_ZONE} --add-port=9443/tcp
    sudo firewall-cmd --runtime-to-permanent
fi

# Generate the certs and the ca bundle
if [ "$SKIP_CERT" = false ] ; then
    mkdir -p ${TMPDIR}
    rm -rf ${TMPDIR}/* || true

    openssl req -newkey rsa:2048 -days 3650 -nodes -x509 \
    -subj "/CN=${HOSTNAME}" \
    -addext "subjectAltName = IP:${CRC_IP}" \
    -keyout ${TMPDIR}/tls.key \
    -out ${TMPDIR}/tls.crt

    cat ${TMPDIR}/tls.crt ${TMPDIR}/tls.key | base64 -w 0 > ${TMPDIR}/bundle.pem
fi

CA_BUNDLE=`cat ${TMPDIR}/bundle.pem`

create_local_webhook() {
    # set the resource type
    RESOURCE_TYPE=$1

    if [[ ${RESOURCE_TYPE} == 'tobiko' ]] ; then
        RESOURCE_TYPE_S="${RESOURCE_TYPE}es"
    else
        RESOURCE_TYPE_S="${RESOURCE_TYPE}s"
    fi

    # Patch the webhook(s)
    cat >> ${TMPDIR}/patch_webhook_configurations.yaml <<EOF_CAT
---
apiVersion: admissionregistration.k8s.io/v1
kind: ValidatingWebhookConfiguration
metadata:
  name: v${RESOURCE_TYPE}.kb.io
webhooks:
- admissionReviewVersions:
  - v1
  clientConfig:
    caBundle: ${CA_BUNDLE}
    url: https://${CRC_IP}:9443/validate-test-openstack-org-v1beta1-${RESOURCE_TYPE}
  failurePolicy: Fail
  matchPolicy: Equivalent
  name: v${RESOURCE_TYPE}.kb.io
  objectSelector: {}
  rules:
  - apiGroups:
    - test.openstack.org
    apiVersions:
    - v1beta1
    operations:
    - CREATE
    - UPDATE
    resources:
    - ${RESOURCE_TYPE_S}
    scope: '*'
  sideEffects: None
  timeoutSeconds: 10
---
apiVersion: admissionregistration.k8s.io/v1
kind: MutatingWebhookConfiguration
metadata:
  name: m${RESOURCE_TYPE}.kb.io
webhooks:
- admissionReviewVersions:
  - v1
  clientConfig:
    caBundle: ${CA_BUNDLE}
    url: https://${CRC_IP}:9443/mutate-test-openstack-org-v1beta1-${RESOURCE_TYPE}
  failurePolicy: Fail
  matchPolicy: Equivalent
  name: m${RESOURCE_TYPE}.kb.io
  objectSelector: {}
  rules:
  - apiGroups:
    - test.openstack.org
    apiVersions:
    - v1beta1
    operations:
    - CREATE
    - UPDATE
    resources:
    - ${RESOURCE_TYPE_S}
    scope: '*'
  sideEffects: None
  timeoutSeconds: 10
EOF_CAT
}

RESOURCE_TYPES=(tempest tobiko ansibletest horizontest)

if [ $# -eq 0 ]; then
    for RESOURCE_TYPE in "${RESOURCE_TYPES[@]}" ; do
        create_local_webhook $RESOURCE_TYPE
    done
elif printf "%s\n" "${RESOURCE_TYPES[@]}" | grep -q "^$1\$" ; then
    # example: ./configure_local_webhook.sh tempest
    create_local_webhook $1
else
    echo "ERROR: No CR matches found! Check your spelling."
    exit 1
fi

oc apply -n openstack -f ${TMPDIR}/patch_webhook_configurations.yaml
