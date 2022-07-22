package objectsusercontroller

import (
	"context"

	pipeline "github.com/ccremer/go-command-pipeline"
	"github.com/crossplane/crossplane-runtime/pkg/errors"
	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	cloudscalev1 "github.com/vshn/provider-cloudscale/apis/cloudscale/v1"
	"github.com/vshn/provider-cloudscale/operator/steps"
	controllerruntime "sigs.k8s.io/controller-runtime"
)

// Delete implements managed.ExternalClient.
func (p *ObjectsUserPipeline) Delete(ctx context.Context, mg resource.Managed) error {
	log := controllerruntime.LoggerFrom(ctx)
	log.Info("Deleting resource")

	user := fromManaged(mg)
	pipe := pipeline.NewPipeline().WithBeforeHooks(steps.DebugLogger(ctx)).
		WithSteps(
			pipeline.NewStepFromFunc("delete objects user", p.deleteObjectsUserFn(user)),
			pipeline.NewStepFromFunc("emit event", p.emitDeletionEventFn(user)),
		)
	result := pipe.RunWithContext(ctx)
	return errors.Wrap(result.Err(), "cannot deprovision objects user")
}

// deleteObjectsUserFn deletes the objects user from the project associated with the API token.
func (p *ObjectsUserPipeline) deleteObjectsUserFn(user *cloudscalev1.ObjectsUser) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		csClient := p.csClient
		log := controllerruntime.LoggerFrom(ctx)

		err := csClient.ObjectsUsers.Delete(ctx, user.Status.AtProvider.UserID)
		if err != nil {
			return err
		}
		log.V(1).Info("Deleted objects user in cloudscale", "userID", user.Status.AtProvider.UserID)
		return nil
	}
}

func (p *ObjectsUserPipeline) emitDeletionEventFn(user *cloudscalev1.ObjectsUser) func(ctx context.Context) error {
	return func(_ context.Context) error {
		p.recorder.Event(user, event.Event{
			Type:    event.TypeNormal,
			Reason:  "Deleted",
			Message: "ObjectsUser deleted",
		})
		return nil
	}
}
