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

# Patch the webhook(s)
cat >> ${TMPDIR}/patch_webhook_configurations.yaml <<EOF_CAT
apiVersion: admissionregistration.k8s.io/v1
kind: ValidatingWebhookConfiguration
metadata:
  name: mtempest.kb.io
webhooks:
- admissionReviewVersions:
  - v1
  clientConfig:
    caBundle: ${CA_BUNDLE}
    url: https://${CRC_IP}:9443/validate-test-openstack-org-v1beta1-tempest
  failurePolicy: Fail
  matchPolicy: Equivalent
  name: mtempest.kb.io
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
    - tempests
    scope: '*'
  sideEffects: None
  timeoutSeconds: 10
EOF_CAT

/home/cloud-user/.crc/cache/crc_libvirt_4.14.7_amd64/oc apply -n openstack -f ${TMPDIR}/patch_webhook_configurations.yaml
