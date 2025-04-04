package project

var (
	description = "app-admission-controller validates and defaults app CRs."
	gitSHA      = "n/a"
	name        = "app-admission-controller" //nolint:gosec
	source      = "https://github.com/giantswarm/app-admission-controller"
	version     = "2.0.1-dev"
)

func Description() string {
	return description
}

func GitSHA() string {
	return gitSHA
}

func Name() string {
	return name
}

func Source() string {
	return source
}

func Version() string {
	return version
}
