#!/usr/bin/env bash

function latestRelease {
  release=$(gh release list --repo $1 -L 1 | head -n 2 | awk '{ print $1; }')
  echo ${release:1}
}

function updateVersions {
  envsubst < hack/versions.tmpl > pkg/config/versions.go
  envsubst < hack/validator.tmpl > tests/integration/validator/testcases/data/validator.yaml
  echo "Updated versions.go & test data with latest validator versions."
}

export AWS_VERSION=$(latestRelease validator-labs/validator-plugin-aws)
export AZURE_VERSION=$(latestRelease validator-labs/validator-plugin-azure)
export KUBESCAPE_VERSION=$(latestRelease validator-labs/validator-plugin-kubescape)
export MAAS_VERSION=$(latestRelease validator-labs/validator-plugin-maas)
export NETWORK_VERSION=$(latestRelease validator-labs/validator-plugin-network)
export OCI_VERSION=$(latestRelease validator-labs/validator-plugin-oci)
export VSPHERE_VERSION=$(latestRelease validator-labs/validator-plugin-vsphere)
export VALIDATOR_VERSION=$(latestRelease validator-labs/validator)

updateVersions