package steps

import (
	"context"
	"fmt"
	pipeline "github.com/ccremer/go-command-pipeline"
	"k8s.io/client-go/tools/record"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// clientKey identifies the Kubernetes client in the context.
type clientKey struct{}

// eventRecorderKey identifies the Kubernetes event recorder in the context.
type eventRecorderKey struct{}

// SetClientInContext sets the given client in the context.
func SetClientInContext(ctx context.Context, c client.Client) {
	pipeline.StoreInContext(ctx, clientKey{}, c)
}

// GetClientFromContext returns the client from the context.
func GetClientFromContext(ctx context.Context) client.Client {
	return GetFromContextOrPanic(ctx, clientKey{}).(client.Client)
}

// SetEventRecorderInContext sets the given recorder in the context.
func SetEventRecorderInContext(ctx context.Context, eventRecorder record.EventRecorder) {
	pipeline.StoreInContext(ctx, eventRecorderKey{}, eventRecorder)
}

// GetEventRecorderFromContext returns the recorder from the context.
func GetEventRecorderFromContext(ctx context.Context) record.EventRecorder {
	return GetFromContextOrPanic(ctx, eventRecorderKey{}).(record.EventRecorder)
}

// GetFromContextOrPanic returns the object if the key exists.
// If the does not exist, then it panics.
// May return nil if the key exists but the value actually is nil.
func GetFromContextOrPanic(ctx context.Context, key any) any {
	val, exists := pipeline.LoadFromContext(ctx, key)
	if !exists {
		keyName := reflect.TypeOf(key).Name()
		panic(fmt.Errorf("key %q does not exist in the given context", keyName))
	}
	return val
}
