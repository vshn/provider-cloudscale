package objectsusercontroller

import (
	"context"
	"fmt"
	"strings"

	pipeline "github.com/ccremer/go-command-pipeline"
	cloudscalesdk "github.com/cloudscale-ch/cloudscale-go-sdk/v2"
	"github.com/crossplane/crossplane-runtime/pkg/errors"
	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	cloudscalev1 "github.com/vshn/provider-cloudscale/apis/cloudscale/v1"
	"github.com/vshn/provider-cloudscale/operator/pipelineutil"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// Create implements managed.ExternalClient.
func (p *ObjectsUserPipeline) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	log := controllerruntime.LoggerFrom(ctx)
	log.Info("Creating resource")

	user := fromManaged(mg)
	if user.Status.AtProvider.UserID != "" {
		// User already exists
		return managed.ExternalCreation{}, nil
	}

	pctx := &pipelineContext{Context: ctx, user: user}
	pipe := pipeline.NewPipeline[*pipelineContext]()
	pipe.WithBeforeHooks(pipelineutil.DebugLogger(pctx)).
		WithSteps(
			pipe.NewStep("create objects user", p.createObjectsUser),
			pipe.When(hasSecretRef,
				"ensure credentials secret", p.ensureCredentialsSecret,
			),
			pipe.NewStep("emit event", p.emitCreationEvent),
		)
	err := pipe.RunWithContext(pctx)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, "cannot create objects user")
	}

	return managed.ExternalCreation{ConnectionDetails: toConnectionDetails(pctx.csUser)}, nil
}

// createObjectsUser creates a new objects user in the project associated with the API token.
func (p *ObjectsUserPipeline) createObjectsUser(ctx *pipelineContext) error {
	csClient := p.csClient
	log := controllerruntime.LoggerFrom(ctx)
	user := ctx.user

	csUser, err := csClient.ObjectsUsers.Create(ctx, &cloudscalesdk.ObjectsUserRequest{
		DisplayName: user.GetDisplayName(),
		TaggedResourceRequest: cloudscalesdk.TaggedResourceRequest{
			Tags: toTagMap(user.Spec.ForProvider.Tags),
		},
	})
	if err != nil {
		return err
	}
	// Limitation by crossplane: The interface managed.ExternalClient doesn't allow updating the resource during creation except annotations.
	// But we need to somehow store the user ID returned by the creation operation, because cloudscale API allows multiple users with the same display name.
	// So we store it in an annotation since that is the only allowed place to update our resource.
	// However, once we observe the spec again, we will copy the user ID from the annotation to the status field,
	//  and that will become the authoritative source of truth for future reconciliations.
	metav1.SetMetaDataAnnotation(&user.ObjectMeta, UserIDAnnotationKey, csUser.ID)

	log.V(1).Info("Created objects user in cloudscale", "userID", csUser.ID, "displayName", csUser.DisplayName, "tags", csUser.Tags)
	ctx.csUser = csUser
	return nil
}

func (p *ObjectsUserPipeline) emitCreationEvent(ctx *pipelineContext) error {
	p.recorder.Event(ctx.user, event.Event{
		Type:    event.TypeNormal,
		Reason:  "Created",
		Message: "ObjectsUser successfully created",
	})
	return nil
}

// ensureCredentialsSecret creates the secret with ObjectsUser's S3 credentials.
// The secret is updated in case the keys change, and an owner reference to the ObjectsUser is set.
func (p *ObjectsUserPipeline) ensureCredentialsSecret(ctx *pipelineContext) error {
	kube := p.kube
	csUser := ctx.csUser
	user := ctx.user
	log := controllerruntime.LoggerFrom(ctx)

	secretRef := user.Spec.WriteConnectionSecretToReference

	ctx.credentialsSecret = &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: secretRef.Name, Namespace: secretRef.Namespace}}
	_, err := controllerruntime.CreateOrUpdate(ctx, kube, ctx.credentialsSecret, func() error {
		secret := ctx.credentialsSecret
		secret.Labels = labels.Merge(secret.Labels, getCommonLabels(user.Name))
		if secret.Data == nil {
			secret.Data = map[string][]byte{}
		}
		for k, v := range toConnectionDetails(csUser) {
			secret.Data[k] = v
		}
		return controllerutil.SetOwnerReference(user, secret, kube.Scheme())
	})
	if err != nil {
		return err
	}
	log.V(1).Info("Ensured credential secret", "secretName", fmt.Sprintf("%s/%s", secretRef.Namespace, secretRef.Name))
	return nil
}

func getCommonLabels(instanceName string) labels.Set {
	// https://kubernetes.io/docs/concepts/overview/working-with-objects/common-labels/
	return labels.Set{
		"app.kubernetes.io/instance":   instanceName,
		"app.kubernetes.io/managed-by": cloudscalev1.Group,
		"app.kubernetes.io/created-by": fmt.Sprintf("controller-%s", strings.ToLower(cloudscalev1.ObjectsUserKind)),
	}
}
