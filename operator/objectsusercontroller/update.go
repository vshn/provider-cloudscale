package objectsusercontroller

import (
	"context"

	pipeline "github.com/ccremer/go-command-pipeline"
	cloudscalesdk "github.com/cloudscale-ch/cloudscale-go-sdk/v2"
	"github.com/crossplane/crossplane-runtime/pkg/errors"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	"github.com/vshn/provider-cloudscale/operator/pipelineutil"
	controllerruntime "sigs.k8s.io/controller-runtime"
)

// Update implements managed.ExternalClient.
func (p *ObjectsUserPipeline) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	log := controllerruntime.LoggerFrom(ctx)
	log.Info("Updating resource")
	user := fromManaged(mg)

	pctx := &pipelineContext{Context: ctx, user: user}
	pipe := pipeline.NewPipeline[*pipelineContext]()
	pipe.WithBeforeHooks(pipelineutil.DebugLogger(pctx)).
		WithSteps(
			pipe.NewStep("update objects user", p.updateObjectsUser),
			pipe.When(hasSecretRef,
				"ensure credentials secret", p.ensureCredentialsSecret,
			),
		)
	err := pipe.RunWithContext(pctx)

	return managed.ExternalUpdate{}, errors.Wrap(err, "cannot update objects user")
}

// updateObjectsUser updates the objects user identified by user ID.
func (p *ObjectsUserPipeline) updateObjectsUser(ctx *pipelineContext) error {
	csClient := p.csClient
	log := controllerruntime.LoggerFrom(ctx)
	user := ctx.user

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
