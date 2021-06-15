// +build k8srequired

package templates

// AppAdmissionControllerValues values required by app-admission-controller chart.
const AppAdmissionControllerValues = `
provider:
  kind: aws
registry:
  domain: quay.io
`
