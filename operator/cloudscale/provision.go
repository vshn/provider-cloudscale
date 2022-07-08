package cloudscale

import (
	"context"
	"fmt"
	pipeline "github.com/ccremer/go-command-pipeline"
	cloudscalev1 "github.com/vshn/appcat-service-s3/apis/cloudscale/v1"
	"github.com/vshn/appcat-service-s3/apis/conditions"
	"github.com/vshn/appcat-service-s3/operator/steps"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/labels"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

// ObjectsUserPipeline provisions ObjectsUsers on cloudscale.ch
type ObjectsUserPipeline struct {
}

// NewObjectsUserPipeline returns a new instance of ObjectsUserPipeline.
func NewObjectsUserPipeline() *ObjectsUserPipeline {
	return &ObjectsUserPipeline{}
}

// Run executes the business logic.
func (p *ObjectsUserPipeline) Run(ctx context.Context) error {
	pipe := pipeline.NewPipeline().WithBeforeHooks(debugLogger(ctx)).
		WithSteps(
			pipeline.NewStepFromFunc("add finalizer", steps.AddFinalizerFn(ObjectsUserKey{}, userFinalizer)),
			pipeline.NewStepFromFunc("create client", CreateCloudscaleClientFn(APIToken)),
			pipeline.IfOrElse(isObjectsUserIDKnown(),
				pipeline.NewStepFromFunc("fetch objects user", GetObjectsUserFn()),
				pipeline.NewPipeline().WithNestedSteps("new user",
					pipeline.NewStepFromFunc("create objects user", CreateObjectsUserFn()),
					pipeline.NewStepFromFunc("set user in status", steps.UpdateStatusFn(ObjectsUserKey{})),
					pipeline.NewStepFromFunc("emit event", emitSuccessEventFn()),
				),
			),
			pipeline.NewStepFromFunc("ensure credential secret", EnsureCredentialSecretFn()),
			pipeline.NewStepFromFunc("set status condition", markUserReadyFn()),
		).
		WithFinalizer(errorHandler())
	result := pipe.RunWithContext(ctx)
	return result.Err()
}

func isObjectsUserIDKnown() func(ctx context.Context) bool {
	return func(ctx context.Context) bool {
		user := steps.GetFromContextOrPanic(ctx, ObjectsUserKey{}).(*cloudscalev1.ObjectsUser)
		return user.Status.UserID != ""
	}
}

func emitSuccessEventFn() func(ctx context.Context) error {
	return func(ctx context.Context) error {
		recorder := steps.GetEventRecorderFromContext(ctx)
		user := steps.GetFromContextOrPanic(ctx, ObjectsUserKey{}).(client.Object)

		recorder.Event(user, v1.EventTypeNormal, "Created", "ObjectsUser successfully created")
		return nil
	}
}

func markUserReadyFn() func(ctx context.Context) error {
	return func(ctx context.Context) error {
		kube := steps.GetClientFromContext(ctx)
		user := steps.GetFromContextOrPanic(ctx, ObjectsUserKey{}).(*cloudscalev1.ObjectsUser)

		meta.SetStatusCondition(&user.Status.Conditions, conditions.Ready())
		meta.RemoveStatusCondition(&user.Status.Conditions, conditions.TypeFailed)
		return kube.Status().Update(ctx, user)
	}
}

func getCommonLabels(instanceName string) labels.Set {
	// https://kubernetes.io/docs/concepts/overview/working-with-objects/common-labels/
	return labels.Set{
		"app.kubernetes.io/instance":   instanceName,
		"app.kubernetes.io/managed-by": cloudscalev1.Group,
		"app.kubernetes.io/created-by": fmt.Sprintf("controller-%s", strings.ToLower(cloudscalev1.ObjectsUserKind)),
	}
}

func debugLogger(ctx context.Context) []pipeline.Listener {
	log := controllerruntime.LoggerFrom(ctx)
	hook := func(step pipeline.Step) {
		log.V(2).Info(`Entering step "` + step.Name + `"`)
	}
	return []pipeline.Listener{hook}
}
