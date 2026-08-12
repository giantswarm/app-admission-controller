//go:build k8srequired
// +build k8srequired

package templates

// CertManagerValues values required by cert-manager-app chart.
const CertManagerValues = `
global:
  podSecurityStandards:
    enforced: false
prometheus:
  servicemonitor:
    enabled: false
`

// AppAdmissionControllerValues values required by app-admission-controller chart.
const AppAdmissionControllerValues = `
provider:
  kind: aws
serviceMonitor:
  enabled: false
psp:
  enableOverrides: false
`
