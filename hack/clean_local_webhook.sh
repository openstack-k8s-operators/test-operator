#!/bin/bash
set -ex

oc delete validatingwebhookconfiguration/vtempest.kb.io --ignore-not-found
oc delete mutatingwebhookconfiguration/mtempest.kb.io --ignore-not-found
oc delete validatingwebhookconfiguration/vtobiko.kb.io --ignore-not-found
oc delete mutatingwebhookconfiguration/mtobiko.kb.io --ignore-not-found
oc delete validatingwebhookconfiguration/vhorizontest.kb.io --ignore-not-found
oc delete mutatingwebhookconfiguration/mhorizontest.kb.io --ignore-not-found
oc delete validatingwebhookconfiguration/vansibletest.kb.io --ignore-not-found
oc delete mutatingwebhookconfiguration/mansibletest.kb.io --ignore-not-found
