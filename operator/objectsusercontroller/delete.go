package objectsusercontroller

import (
	"context"

	pipeline "github.com/ccremer/go-command-pipeline"
	"github.com/crossplane/crossplane-runtime/pkg/errors"
	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	"github.com/vshn/provider-cloudscale/operator/pipelineutil"
	controllerruntime "sigs.k8s.io/controller-runtime"
)

// Delete implements managed.ExternalClient.
func (p *ObjectsUserPipeline) Delete(ctx context.Context, mg resource.Managed) error {
	log := controllerruntime.LoggerFrom(ctx)
	log.Info("Deleting resource")

	user := fromManaged(mg)
	pctx := &pipelineContext{Context: ctx, user: user}
	pipe := pipeline.NewPipeline[*pipelineContext]()
	pipe.WithBeforeHooks(pipelineutil.DebugLogger(pctx)).
		WithSteps(
			pipe.NewStep("delete objects user", p.deleteObjectsUser),
			pipe.NewStep("emit event", p.emitDeletionEvent),
		)
	err := pipe.RunWithContext(pctx)
	return errors.Wrap(err, "cannot deprovision objects user")
}

// deleteObjectsUser deletes the objects user from the project associated with the API token.
func (p *ObjectsUserPipeline) deleteObjectsUser(ctx *pipelineContext) error {
	csClient := p.csClient
	log := controllerruntime.LoggerFrom(ctx)
	user := ctx.user

	err := csClient.ObjectsUsers.Delete(ctx, user.Status.AtProvider.UserID)
	if err != nil {
		return err
	}
	log.V(1).Info("Deleted objects user in cloudscale", "userID", user.Status.AtProvider.UserID)
	return nil
}

func (p *ObjectsUserPipeline) emitDeletionEvent(ctx *pipelineContext) error {
	p.recorder.Event(ctx.user, event.Event{
		Type:    event.TypeNormal,
		Reason:  "Deleted",
		Message: "ObjectsUser deleted",
	})
	return nil
}
