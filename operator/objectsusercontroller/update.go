package objectsusercontroller

import (
	"context"

	pipeline "github.com/ccremer/go-command-pipeline"
	cloudscalesdk "github.com/cloudscale-ch/cloudscale-go-sdk/v2"
	"github.com/crossplane/crossplane-runtime/pkg/errors"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	cloudscalev1 "github.com/vshn/provider-cloudscale/apis/cloudscale/v1"
	"github.com/vshn/provider-cloudscale/operator/steps"
	controllerruntime "sigs.k8s.io/controller-runtime"
)

// Update implements managed.ExternalClient.
func (p *ObjectsUserPipeline) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	log := controllerruntime.LoggerFrom(ctx)
	log.Info("Updating resource")
	user := fromManaged(mg)

	pipe := pipeline.NewPipeline().WithBeforeHooks(steps.DebugLogger(ctx)).
		WithSteps(
			pipeline.NewStepFromFunc("update objects user", p.updateObjectsUserFn(user)),
			pipeline.If(hasSecretRef(user),
				pipeline.NewStepFromFunc("ensure credentials secret", p.ensureCredentialsSecretFn(user)),
			),
			pipeline.NewStepFromFunc("emit event", p.emitDeletionEventFn(user)),
		)
	result := pipe.RunWithContext(ctx)

	return managed.ExternalUpdate{}, errors.Wrap(result.Err(), "cannot update objects user")
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
