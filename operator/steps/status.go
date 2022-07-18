package steps

import (
	"context"

	"github.com/vshn/provider-cloudscale/apis/conditions"
	"k8s.io/apimachinery/pkg/api/meta"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// UpdateStatusFn returns a func that updates the status of the object identified by key retrieved from the context.
func UpdateStatusFn(objKey any) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		kube := GetClientFromContext(ctx)
		obj := GetFromContextOrPanic(ctx, objKey).(client.Object)

		return kube.Status().Update(ctx, obj)
	}
}

// MarkObjectReadyFn updates the resource identified by objKey with following conditions:
//  Ready: True
//  Failed: <Removed>
// The resource must implement conditions.ObjectWithConditions interface.
func MarkObjectReadyFn(objKey any) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		kube := GetClientFromContext(ctx)
		obj := GetFromContextOrPanic(ctx, objKey).(conditions.ObjectWithConditions)

		conds := obj.GetConditions()

		meta.SetStatusCondition(&conds, conditions.Ready())
		meta.RemoveStatusCondition(&conds, conditions.TypeFailed)
		obj.SetConditions(conds)
		return kube.Status().Update(ctx, obj)
	}
}
