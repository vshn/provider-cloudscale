package steps

import (
	"context"
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
