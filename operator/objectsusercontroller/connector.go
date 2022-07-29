package objectsusercontroller

import (
	"context"
	"fmt"

	pipeline "github.com/ccremer/go-command-pipeline"
	cloudscalesdk "github.com/cloudscale-ch/cloudscale-go-sdk/v2"
	"github.com/crossplane/crossplane-runtime/pkg/errors"
	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	cloudscalev1 "github.com/vshn/provider-cloudscale/apis/cloudscale/v1"
	providerv1 "github.com/vshn/provider-cloudscale/apis/provider/v1"
	"github.com/vshn/provider-cloudscale/operator/steps"
	"golang.org/x/oauth2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// CloudscaleAPITokenKey identifies the key in which the API token of the cloudscale.ch API is expected in a Secret.
	CloudscaleAPITokenKey = "CLOUDSCALE_API_TOKEN"
)

type objectsUserConnector struct {
	kube     client.Client
	recorder event.Recorder
}

type providerConfigKey struct{}
type apiTokenKey struct{}

// Connect implements managed.ExternalConnecter.
func (c *objectsUserConnector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	ctx = pipeline.MutableContext(ctx)
	log := ctrl.LoggerFrom(ctx)
	log.V(1).Info("Connecting resource")

	user := fromManaged(mg)
	result := pipeline.NewPipeline().WithBeforeHooks(steps.DebugLogger(ctx)).WithSteps(
		pipeline.NewStepFromFunc("track provider config", c.trackProviderConfigFn(user)),
		pipeline.NewStepFromFunc("fetch provider config", c.fetchProviderConfigFn(user)),
		pipeline.NewStepFromFunc("fetch API token", c.fetchApiToken),
	).RunWithContext(ctx)
	if result.IsFailed() {
		return nil, result.Err()
	}
	csClient := c.createCloudscaleClient(ctx)
	return NewPipeline(c.kube, c.recorder, csClient), nil
}

// createCloudscaleClient creates a new kube using the API token provided.
func (c *objectsUserConnector) createCloudscaleClient(ctx context.Context) *cloudscalesdk.Client {
	token := pipeline.MustLoadFromContext(ctx, apiTokenKey{}).(string)
	tc := oauth2.NewClient(ctx, oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token}))
	csClient := cloudscalesdk.NewClient(tc)
	return csClient
}

func (c *objectsUserConnector) fetchProviderConfigFn(user *cloudscalev1.ObjectsUser) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		config := &providerv1.ProviderConfig{}
		err := c.kube.Get(ctx, types.NamespacedName{Name: user.Spec.ProviderConfigReference.Name}, config)
		pipeline.StoreInContext(ctx, providerConfigKey{}, config)
		return errors.Wrap(err, "cannot get ProviderConfig")
	}
}

func (c *objectsUserConnector) fetchApiToken(ctx context.Context) error {
	providerConfig := pipeline.MustLoadFromContext(ctx, providerConfigKey{}).(*providerv1.ProviderConfig)
	secretRef := providerConfig.Spec.Credentials.APITokenSecretRef
	secret := &corev1.Secret{}
	err := c.kube.Get(ctx, types.NamespacedName{Name: secretRef.Name, Namespace: secretRef.Namespace}, secret)
	if err != nil {
		return errors.Wrap(err, "cannot get secret with API token")
	}
	if value, exists := secret.Data[CloudscaleAPITokenKey]; exists && string(value) != "" {
		pipeline.StoreInContext(ctx, apiTokenKey{}, string(value))
		return nil
	}
	return fmt.Errorf("%s doesn't exist in secret %s/%s", CloudscaleAPITokenKey, secret.Namespace, secret.Name)
}

// trackProviderConfigFn ensures that the ProviderConfig referenced by the ObjectsUser is not deleted until all ObjectsUser stop using the ProviderConfig.
// It's similar to a finalizer: Without the ProviderConfig we can't deprovision ObjectsUser (missing credentials).
// It returns an error if the ProviderConfig reference is missing.
func (c *objectsUserConnector) trackProviderConfigFn(user *cloudscalev1.ObjectsUser) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		u := resource.NewProviderConfigUsageTracker(c.kube, &providerv1.ProviderConfigUsage{})
		err := u.Track(ctx, user)
		return err
	}
}
