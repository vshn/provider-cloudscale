package cloudscale

import (
	"context"
	"fmt"
	"strings"

	pipeline "github.com/ccremer/go-command-pipeline"
	cloudscalev1 "github.com/vshn/appcat-service-s3/apis/cloudscale/v1"
	"github.com/vshn/appcat-service-s3/apis/conditions"
	"github.com/vshn/appcat-service-s3/operator/steps"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
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
	pipe := pipeline.NewPipeline().WithBeforeHooks(steps.DebugLogger(ctx)).
		WithSteps(
			pipeline.NewStepFromFunc("add finalizer", steps.AddFinalizerFn(ObjectsUserKey{}, userFinalizer)),
			pipeline.NewStepFromFunc("create client", CreateCloudscaleClientFn(APIToken)),
			pipeline.IfOrElse(isObjectsUserIDKnown,
				pipeline.NewStepFromFunc("fetch objects user", GetObjectsUser),
				pipeline.NewPipeline().WithNestedSteps("new user",
					pipeline.NewStepFromFunc("create objects user", CreateObjectsUser),
					pipeline.NewStepFromFunc("set user in status", steps.UpdateStatusFn(ObjectsUserKey{})),
					pipeline.NewStepFromFunc("emit event", emitSuccessEvent),
				),
			),
			pipeline.NewStepFromFunc("ensure credential secret", EnsureCredentialSecret),
			pipeline.NewStepFromFunc("set status condition", steps.MarkObjectReadyFn(ObjectsUserKey{})),
		).
		WithFinalizer(steps.ErrorHandlerFn(ObjectsUserKey{}, conditions.ReasonProvisioningFailed))
	result := pipe.RunWithContext(ctx)
	return result.Err()
}

func isObjectsUserIDKnown(ctx context.Context) bool {
	user := steps.GetFromContextOrPanic(ctx, ObjectsUserKey{}).(*cloudscalev1.ObjectsUser)
	return user.Status.UserID != ""
}

func emitSuccessEvent(ctx context.Context) error {
	recorder := steps.GetEventRecorderFromContext(ctx)
	user := steps.GetFromContextOrPanic(ctx, ObjectsUserKey{}).(client.Object)

	recorder.Event(user, v1.EventTypeNormal, "Created", "ObjectsUser successfully created")
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
