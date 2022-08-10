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
	"github.com/vshn/provider-cloudscale/operator/pipelineutil"
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

type connectContext struct {
	context.Context
	user           *cloudscalev1.ObjectsUser
	providerConfig *providerv1.ProviderConfig
	apiTokenSecret *corev1.Secret
	apiToken       string
}

// Connect implements managed.ExternalConnecter.
func (c *objectsUserConnector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	log := ctrl.LoggerFrom(ctx)
	log.V(1).Info("Connecting resource")

	user := fromManaged(mg)
	pctx := &connectContext{Context: ctx, user: user}
	p := pipeline.NewPipeline[*connectContext]()
	err := p.WithBeforeHooks(pipelineutil.DebugLogger(pctx)).
		WithSteps(
			p.NewStep("track provider config", c.trackProviderConfig),
			p.NewStep("fetch provider config", c.fetchProviderConfig),
			p.NewStep("fetch API token", c.fetchApiTokenSecret),
			p.NewStep("read API token", c.readApiToken),
		).RunWithContext(pctx)
	if err != nil {
		return nil, err
	}
	csClient := c.createCloudscaleClient(pctx)
	return NewPipeline(c.kube, c.recorder, csClient), nil
}

// createCloudscaleClient creates a new kube using the API token provided.
func (c *objectsUserConnector) createCloudscaleClient(ctx *connectContext) *cloudscalesdk.Client {
	token := ctx.apiToken
	tc := oauth2.NewClient(ctx, oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token}))
	csClient := cloudscalesdk.NewClient(tc)
	return csClient
}

func (c *objectsUserConnector) fetchProviderConfig(ctx *connectContext) error {
	config := &providerv1.ProviderConfig{}
	err := c.kube.Get(ctx, types.NamespacedName{Name: ctx.user.Spec.ProviderConfigReference.Name}, config)
	ctx.providerConfig = config
	return errors.Wrap(err, "cannot get ProviderConfig")
}

func (c *objectsUserConnector) fetchApiTokenSecret(ctx *connectContext) error {
	providerConfig := ctx.providerConfig
	secretRef := providerConfig.Spec.Credentials.APITokenSecretRef
	secret := &corev1.Secret{}
	err := c.kube.Get(ctx, types.NamespacedName{Name: secretRef.Name, Namespace: secretRef.Namespace}, secret)
	ctx.apiTokenSecret = secret
	return errors.Wrap(err, "cannot get secret with API token")
}

func (c *objectsUserConnector) readApiToken(ctx *connectContext) error {
	secret := ctx.apiTokenSecret
	if value, exists := secret.Data[CloudscaleAPITokenKey]; exists && string(value) != "" {
		ctx.apiToken = string(value)
		return nil
	}
	return fmt.Errorf("%s doesn't exist in secret %s/%s", CloudscaleAPITokenKey, secret.Namespace, secret.Name)
}

// trackProviderConfig ensures that the ProviderConfig referenced by the ObjectsUser is not deleted until all ObjectsUser stop using the ProviderConfig.
// It's similar to a finalizer: Without the ProviderConfig we can't deprovision ObjectsUser (missing credentials).
// It returns an error if the ProviderConfig reference is missing.
func (c *objectsUserConnector) trackProviderConfig(ctx *connectContext) error {
	u := resource.NewProviderConfigUsageTracker(c.kube, &providerv1.ProviderConfigUsage{})
	err := u.Track(ctx, ctx.user)
	return err
}
