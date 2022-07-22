package cloudscale

import (
	"context"
	"fmt"

	pipeline "github.com/ccremer/go-command-pipeline"
	cloudscalev1 "github.com/vshn/provider-cloudscale/apis/cloudscale/v1"
	"github.com/vshn/provider-cloudscale/apis/conditions"
	"github.com/vshn/provider-cloudscale/operator/steps"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// DeletionPipeline deletes ObjectsUsers on cloudscale.ch
type DeletionPipeline struct{}

// NewDeletionPipeline returns a new instance of DeletionPipeline.
func NewDeletionPipeline() *DeletionPipeline {
	return &DeletionPipeline{}
}

// Run executes the business logic.
func (p *DeletionPipeline) Run(ctx context.Context) error {
	pipe := pipeline.NewPipeline().WithBeforeHooks(steps.DebugLogger(ctx)).
		WithSteps(
			pipeline.If(isObjectsUserIDKnown, pipeline.NewPipeline().WithNestedSteps("deprovision objects user",
				pipeline.NewStepFromFunc("create client", CreateCloudscaleClientFn(APIToken)),
				pipeline.NewStepFromFunc("delete objects user", DeleteObjectsUser),
				// Note: We do not need to check if there are Bucket resources still requiring the Secret.
				// Cloudscale's API returns an error if there are still buckets existing for that user, which ultimately also ends up as a Failed condition in the ObjectsUser resource.
				pipeline.NewStepFromFunc("delete finalizer from secret", deleteFinalizerFromSecret),
				pipeline.NewStepFromFunc("emit event", emitDeletionEvent),
			)),
			pipeline.NewStepFromFunc("remove finalizer", steps.RemoveFinalizerFn(ObjectsUserKey{}, userFinalizer)),
		).
		WithFinalizer(steps.ErrorHandlerFn(ObjectsUserKey{}, conditions.ReasonDeletionFailed))
	result := pipe.RunWithContext(ctx)
	return result.Err()
}

func deleteFinalizerFromSecret(ctx context.Context) error {
	kube := steps.GetClientFromContext(ctx)
	user := steps.GetFromContextOrPanic(ctx, ObjectsUserKey{}).(*cloudscalev1.ObjectsUser)
	log := controllerruntime.LoggerFrom(ctx)

	secret := &corev1.Secret{}
	name := user.Spec.ForProvider.SecretRef.Name
	namespace := user.Spec.ForProvider.SecretRef.Namespace
	err := kube.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, secret)
	if apierrors.IsNotFound(err) {
		return nil // doesn't exist anymore, ignore
	}
	if err != nil {
		return err // some other error
	}
	err = steps.RemoveFinalizerFn(ObjectsUserKey{}, userFinalizer)(ctx)
	return logIfNotError(err, log, 1, "Deleted finalizer from credentials secret", "secretName", fmt.Sprintf("%s/%s", namespace, name))
}

func emitDeletionEvent(ctx context.Context) error {
	recorder := steps.GetEventRecorderFromContext(ctx)
	user := steps.GetFromContextOrPanic(ctx, ObjectsUserKey{}).(client.Object)

	recorder.Event(user, corev1.EventTypeNormal, "Deleted", "ObjectsUser deleted")
	return nil
}
