package objectsusercontroller

import (
	"context"
	"fmt"

	pipeline "github.com/ccremer/go-command-pipeline"
	cloudscalesdk "github.com/cloudscale-ch/cloudscale-go-sdk/v2"
	"github.com/crossplane/crossplane-runtime/pkg/errors"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	cloudscalev1 "github.com/vshn/provider-cloudscale/apis/cloudscale/v1"
	"github.com/vshn/provider-cloudscale/operator/steps"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Update implements managed.ExternalClient.
func (p *ObjectsUserPipeline) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	log := controllerruntime.LoggerFrom(ctx)
	log.Info("Updating resource")
	user := fromManaged(mg)

	pipe := pipeline.NewPipeline().WithBeforeHooks(steps.DebugLogger(ctx)).
		WithSteps(
			pipeline.NewStepFromFunc("update objects user", p.updateObjectsUserFn(user)),
			pipeline.IfOrElse(hasSecretRef(user),
				pipeline.NewStepFromFunc("ensure credentials secret", p.ensureCredentialsSecretFn(user)),
				pipeline.NewPipeline().WithNestedSteps("deprovision credentials secret",
					pipeline.NewStepFromFunc("fetch credentials secret", p.fetchCredentialsSecretFn(user)),
					pipeline.NewStepFromFunc("check ownership", p.checkOwnershipFn(user)),
					pipeline.NewStepFromFunc("delete credentials secret", p.deleteCredentialsSecret),
				).WithErrorHandler(ignoreNotFound),
			),
			pipeline.NewStepFromFunc("emit event", p.emitDeletionEventFn(user)),
		)
	result := pipe.RunWithContext(ctx)

	return managed.ExternalUpdate{}, errors.Wrap(result.Err(), "cannot update objects user")
}

func ignoreNotFound(_ context.Context, err error) error {
	return client.IgnoreNotFound(err)
}

// updateObjectsUserFn updates the objects user identified by user ID.
func (p *ObjectsUserPipeline) updateObjectsUserFn(user *cloudscalev1.ObjectsUser) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		csClient := p.csClient
		log := controllerruntime.LoggerFrom(ctx)

		err := csClient.ObjectsUsers.Update(ctx, user.Status.AtProvider.UserID, &cloudscalesdk.ObjectsUserRequest{
			DisplayName:           user.GetDisplayName(),
			TaggedResourceRequest: cloudscalesdk.TaggedResourceRequest{Tags: toTagMap(user.Spec.ForProvider.Tags)},
		})
		if err != nil {
			return err
		}
		log.V(1).Info("Updated objects user in cloudscale",
			"userID", user.Status.AtProvider.UserID, "displayName", user.GetDisplayName(), "tags", user.Spec.ForProvider.Tags)
		return nil
	}
}

func (p *ObjectsUserPipeline) deleteCredentialsSecret(ctx context.Context) error {
	kube := p.kube
	err := kube.Delete(ctx, p.credentialsSecret)
	return err
}

func (p *ObjectsUserPipeline) fetchCredentialsSecretFn(user *cloudscalev1.ObjectsUser) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		kube := p.kube
		secretRef := user.Spec.ForProvider.SecretRef
		p.credentialsSecret = &corev1.Secret{}

		err := kube.Get(ctx, types.NamespacedName{Namespace: secretRef.Namespace, Name: secretRef.Name}, p.credentialsSecret)
		return err
	}
}

func (p *ObjectsUserPipeline) checkOwnershipFn(user *cloudscalev1.ObjectsUser) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		for _, owner := range p.credentialsSecret.OwnerReferences {
			if owner.APIVersion == cloudscalev1.ObjectsUserGroupVersionKind.GroupVersion().String() && owner.Name == user.Name && owner.Kind == user.Kind {
				return nil
			}
		}
		return fmt.Errorf("user %s doesn't own secret %s/%s", user.Name, p.credentialsSecret.Namespace, p.credentialsSecret.Name)
	}
}
