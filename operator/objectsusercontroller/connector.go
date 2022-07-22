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
	c.user = user
	result := pipeline.NewPipeline().WithBeforeHooks(steps.DebugLogger(ctx)).WithSteps(
		pipeline.NewStepFromFunc("fetch provider config", c.fetchProviderConfig),
		pipeline.NewStepFromFunc("fetch API token secret", c.fetchApiTokenSecret),
		pipeline.NewStepFromFunc("read API token", c.readApiToken),
	).RunWithContext(ctx)
	if result.IsFailed() {
		return nil, result.Err()
	}
	csClient := c.createCloudscaleClient(ctx)
	return NewPipeline(c.kube, c.recorder, csClient), nil
}

// createCloudscaleClient creates a new kube using the API token provided.
func (c *objectsUserConnector) createCloudscaleClient(ctx context.Context) *cloudscalesdk.Client {
	tc := oauth2.NewClient(ctx, oauth2.StaticTokenSource(&oauth2.Token{AccessToken: c.apiToken}))
	csClient := cloudscalesdk.NewClient(tc)
	return csClient
}

func (c *objectsUserConnector) fetchProviderConfig(ctx context.Context) error {
	providerConfigName := c.user.GetProviderConfigName()
	if providerConfigName == "" {
		return fmt.Errorf(".spec.providerConfigRef.Name is required")
	}
	c.providerConfig = &providerv1.ProviderConfig{}
	err := c.kube.Get(ctx, types.NamespacedName{Name: providerConfigName}, c.providerConfig)
	return errors.Wrap(err, "cannot get ProviderConfig")
}

func (c *objectsUserConnector) fetchApiTokenSecret(ctx context.Context) error {
	secretRef := c.providerConfig.Spec.Credentials.APITokenSecretRef
	c.apiTokenSecret = &corev1.Secret{}
	err := c.kube.Get(ctx, types.NamespacedName{Name: secretRef.Name, Namespace: secretRef.Namespace}, c.apiTokenSecret)
	return errors.Wrap(err, "cannot get secret with API token")
}

func (c *objectsUserConnector) readApiToken(_ context.Context) error {
	secret := c.apiTokenSecret
	if value, exists := secret.Data[CloudscaleAPITokenKey]; exists && string(value) != "" {
		c.apiToken = string(value)
		return nil
	}
	return fmt.Errorf("%s doesn't exist in secret %s/%s", CloudscaleAPITokenKey, secret.Namespace, secret.Name)
}
