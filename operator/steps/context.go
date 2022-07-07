package steps

import (
	"context"
	"fmt"
	pipeline "github.com/ccremer/go-command-pipeline"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// clientKey identifies the Kubernetes client in the context.
type clientKey struct{}

// SetClientInContext sets the given client in the context.
func SetClientInContext(ctx context.Context, c client.Client) {
	pipeline.StoreInContext(ctx, clientKey{}, c)
}

// GetClientFromContext returns the client from the context.
func GetClientFromContext(ctx context.Context) client.Client {
	return GetFromContextOrPanic(ctx, clientKey{}).(client.Client)
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
