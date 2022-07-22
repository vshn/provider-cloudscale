package steps

import (
	"context"

	pipeline "github.com/ccremer/go-command-pipeline"
	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/vshn/provider-cloudscale/apis/conditions"
	v1 "k8s.io/api/core/v1"
	controllerruntime "sigs.k8s.io/controller-runtime"
)

// ErrorHandlerFn returns a pipeline.ResultHandler that, if error is non-nil,
//  - sets the Failed condition with the error message
//  - sets the Ready condition to False
//  - updates the status on the resource (if updating status fails, the error will only be logged, not bubbled up)
//  - emits a warning event with the error message
// The object must implement conditions.ObjectWithConditions.
func ErrorHandlerFn(objKey any, reason string) pipeline.ResultHandler {
	return func(ctx context.Context, result pipeline.Result) error {
		if result.IsSuccessful() {
			return nil
		}
		kube := GetClientFromContext(ctx)
		obj := GetFromContextOrPanic(ctx, objKey).(conditions.ObjectWithConditions)
		log := controllerruntime.LoggerFrom(ctx)
		recorder := GetEventRecorderFromContext(ctx)

		conds := obj.GetConditions()
		failedCond := conditions.Failed(result.Err())
		failedCond.Reason = xpv1.ConditionReason(reason)

		// meta.SetStatusCondition(&conds, conditions.NotReady())
		// meta.SetStatusCondition(&conds, failedCond)
		obj.SetConditions(conds)
		err := kube.Status().Update(ctx, obj)
		if err != nil {
			log.V(1).Error(err, "updating status failed")
		}
		recorder.Event(obj, v1.EventTypeWarning, "Failed", result.Err().Error())
		return result.Err()
	}
}
