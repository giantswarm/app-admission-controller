// +build k8srequired

package templates

// AppAdmissionControllerValues values required by app-admission-controller chart.
const AppAdmissionControllerValues = `Installation:
  V1:
    Provider:
      Kind: aws
    Registry:
      Domain: quay.io
`
