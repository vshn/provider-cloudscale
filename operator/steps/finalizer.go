package steps

import (
	"context"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// AddFinalizerFn returns a func that adds the given finalizer to an object identified by `objKey` in the context.
// If the finalizer is already present, this step is a no-op.
// The object from context needs to be of a client.Object.
func AddFinalizerFn(objKey any, finalizer string) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		kube := GetClientFromContext(ctx)
		obj := GetFromContextOrPanic(ctx, objKey).(client.Object)

		if controllerutil.AddFinalizer(obj, finalizer) {
			return kube.Update(ctx, obj)
		}
		return nil
	}
}

// RemoveFinalizerFn returns a func that removes the given finalizer from the object identified by `objKey` in the context.
// If the finalizer is not present, this step is a no-op.
// The object from context needs to be of a client.Object.
func RemoveFinalizerFn(objKey any, finalizer string) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		kube := GetClientFromContext(ctx)
		instance := GetFromContextOrPanic(ctx, objKey).(client.Object)

		if controllerutil.RemoveFinalizer(instance, finalizer) {
			return kube.Update(ctx, instance)
		}
		return nil
	}
}
