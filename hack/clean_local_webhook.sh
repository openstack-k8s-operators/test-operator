#!/bin/bash
set -ex

# Tempest
oc delete validatingwebhookconfiguration/vtempest.kb.io --ignore-not-found
oc delete mutatingwebhookconfiguration/mtempest.kb.io --ignore-not-found

# Tobiko
oc delete validatingwebhookconfiguration/vtobiko.kb.io --ignore-not-found
oc delete mutatingwebhookconfiguration/mtobiko.kb.io --ignore-not-found