#!/bin/bash
set -ex

oc delete validatingwebhookconfiguration/vtempest.kb.io --ignore-not-found
oc delete mutatingwebhookconfiguration/mtempest.kb.io --ignore-not-found
