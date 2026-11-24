package inspector

import (
	"context"
	"strings"

	"github.com/giantswarm/apiextensions-application/api/v1alpha1"
	"github.com/giantswarm/app/v8/pkg/key"
	"github.com/giantswarm/microerror"
	authv1 "k8s.io/api/authentication/v1"
)

const (
	appNotAllowedTemplate       = "installing %#q app is not allowed"
	referenceNotAllowedTemplate = "references to %#q namespace not allowed"
)

// Inspect check App CR against two things at the moment:
// - references to blacklisted namespaces in config or userConfig
// - blacklisted apps being requested, from any catalog
func (i *Inspector) Inspect(ctx context.Context, app v1alpha1.App, userInfo authv1.UserInfo) error {
	var err error

	// Skip early for whitelisted actors
	if i.isWhitelistedActor(ctx, userInfo) {
		i.logger.Debugf(ctx, "skipping validation due to whitelisted user %#q", userInfo.Username)
		return nil
	}

	// Skip early for apps coming from one of
	// blacklisted namespaces, as it means this apps
	// is one of the MAPI apps
	if i.isPrivateApp(ctx, app) {
		i.logger.Debugf(ctx, "skipping validation for app comming from private %#q namespace", app.Namespace)
		return nil
	}

	// Validate app against blacklisted apps
	err = i.isBlacklistedApp(ctx, app)
	if err != nil {
		i.logger.Errorf(ctx, err, "rejecting app %#q in namespace %#q for blacklisted app %#q", app.Name, app.Namespace, key.AppName(app))
		return microerror.Mask(err)
	}

	// Validate app against references to
	// blacklisted namespaces
	err = i.hasBlacklistedReference(ctx, app)
	if err != nil {
		i.logger.Errorf(ctx, err, "rejecting %#q app in %#q namespace due to blacklisted references", app.Name, app.Namespace)
		return microerror.Mask(err)
	}

	return nil
}

// hasBlacklistedReference checks if application references configuration
// from protected namespaces when configuring their unique App CRs, see:
// https://github.com/giantswarm/giantswarm/issues/21953
func (i *Inspector) hasBlacklistedReference(ctx context.Context, app v1alpha1.App) error {
	referencedNamespaces := []string{
		key.AppConfigMapNamespace(app),
		key.AppSecretNamespace(app),
		key.UserConfigMapNamespace(app),
		key.UserSecretNamespace(app),
		key.KubeConfigSecretNamespace(app),
	}

	for _, extraConfig := range key.ExtraConfigs(app) {
		referencedNamespaces = append(referencedNamespaces, extraConfig.Namespace)
	}

	for _, ns := range referencedNamespaces {
		if _, ok := i.fixedNamespaceBlacklist[ns]; ok {
			return microerror.Maskf(securityViolationError, referenceNotAllowedTemplate, ns)
		}
	}

	for _, rns := range referencedNamespaces {
		for _, pns := range i.dynamicNamespaceBlacklist {
			if strings.HasSuffix(rns, pns) || strings.HasPrefix(rns, pns) {
				return microerror.Maskf(securityViolationError, referenceNotAllowedTemplate, rns)
			}
		}
	}

	return nil
}

// isBlacklistedApp checks if app is one of the prohibited apps, see:
// https://github.com/giantswarm/giantswarm/issues/21953
//
// The app name alone decides, whatever catalog the App CR names.
func (i *Inspector) isBlacklistedApp(ctx context.Context, app v1alpha1.App) error {
	if _, ok := i.appBlacklist[key.AppName(app)]; ok {
		return microerror.Maskf(securityViolationError, appNotAllowedTemplate, key.AppName(app))
	}

	return nil
}

// isPrivateApp checks if app comes from one of protected namespaces.
// If yes, it means app is being installed by Giantswarm stuff or
// Giantswarm controllers.
func (i *Inspector) isPrivateApp(ctx context.Context, app v1alpha1.App) bool {
	_, ok := i.fixedNamespaceBlacklist[app.Namespace]
	return ok
}

// isWhitelistedActor checks if request comes from a whitelisted user.
// If yes, it means there is work being done by the Giantswarm
// operators, or by the Giantswarm stuff.
func (i *Inspector) isWhitelistedActor(ctx context.Context, userInfo authv1.UserInfo) bool {
	// Check against user name
	if i.isWhitelistedUser(userInfo.Username) {
		return true
	}

	// Check against user groups
	for _, group := range userInfo.Groups {
		if i.isWhitelistedGroup(group) {
			return true
		}
	}

	return false
}

func (i *Inspector) isWhitelistedGroup(name string) bool {
	_, ok := i.groupWhitelist[name]
	return ok
}

func (i *Inspector) isWhitelistedUser(name string) bool {
	for _, user := range i.userWhitelist {
		if strings.HasPrefix(name, user) {
			return true
		}
	}

	return false
}
