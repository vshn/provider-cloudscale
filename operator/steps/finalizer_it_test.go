//go:build integration

package steps

import (
	"context"
	"testing"

	pipeline "github.com/ccremer/go-command-pipeline"
	"github.com/stretchr/testify/suite"
	"github.com/vshn/provider-cloudscale/operator/operatortest"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type FinalizerSuite struct {
	operatortest.Suite
}

func TestFinalizerSuite(t *testing.T) {
	suite.Run(t, new(FinalizerSuite))
}

func (ts *FinalizerSuite) BeforeTest(suiteName, testName string) {
	ts.Context = pipeline.MutableContext(context.Background())
	SetClientInContext(ts.Context, ts.Client)
}

func (ts *FinalizerSuite) Test_AddFinalizer() {
	tests := map[string]struct {
		prepare        func(resource client.Object)
		givenName      string
		givenNamespace string
		assert         func(previousResource, resource client.Object)
	}{
		"GivenResourceWithoutFinalizer_WhenAddingFinalizer_ThenExpectResourceUpdatedWithAddedFinalizer": {
			prepare: func(resource client.Object) {
				ts.EnsureNS("add-finalizer")
				ts.EnsureResources(resource)
				ts.Assert().Empty(resource.GetFinalizers())
			},

			givenName:      "has-finalizer",
			givenNamespace: "add-finalizer",
			assert: func(previousResource, result client.Object) {
				ts.Require().Len(result.GetFinalizers(), 1, "amount of finalizers")
				ts.Assert().Equal("domain.io/finalizer", result.GetFinalizers()[0])
				ts.Assert().NotEqual(previousResource.GetResourceVersion(), result.GetResourceVersion(), "resource version should change")
			},
		},
		"GivenResourceWithExistingFinalizer_WhenAddingFinalizer_ThenExpectResourceUnchanged": {
			prepare: func(resource client.Object) {
				resource.SetFinalizers([]string{"domain.io/finalizer"})
				ts.EnsureNS("add-finalizer")
				ts.EnsureResources(resource)
			},

			givenName:      "no-finalizer",
			givenNamespace: "add-finalizer",
			assert: func(previousResource, result client.Object) {
				ts.Require().Len(result.GetFinalizers(), 1, "amount of finalizers")
				ts.Assert().Equal("domain.io/finalizer", result.GetFinalizers()[0])
				ts.Assert().Equal(previousResource.GetResourceVersion(), result.GetResourceVersion(), "resource version should be equal")
			},
		},
	}
	for name, tc := range tests {
		ts.Run(name, func() {
			type resourceKey struct{}
			// Arrange
			resource := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{Name: tc.givenName, Namespace: tc.givenNamespace},
			}
			pipeline.StoreInContext(ts.Context, resourceKey{}, resource)
			tc.prepare(resource)
			previousVersion := resource.DeepCopy()

			// Act
			err := AddFinalizerFn(resourceKey{}, "domain.io/finalizer")(ts.Context)
			ts.Require().NoError(err)

			// Assert
			result := &corev1.ConfigMap{}
			ts.FetchResource(client.ObjectKeyFromObject(resource), result)
			tc.assert(previousVersion, result)
		})
	}
}

func (ts *FinalizerSuite) Test_RemoveFinalizer() {
	tests := map[string]struct {
		prepare        func(resource client.Object)
		givenName      string
		givenNamespace string
		assert         func(previousResource, resource client.Object)
	}{
		"GivenResourceWithFinalizer_WhenDeletingFinalizer_ThenExpectResourceUpdatedWithRemovedFinalizer": {
			prepare: func(resource client.Object) {
				resource.SetFinalizers([]string{"domain.io/finalizer"})
				ts.EnsureNS("remove-finalizer")
				ts.EnsureResources(resource)
				ts.Assert().NotEmpty(resource.GetFinalizers())
			},

			givenName:      "has-finalizer",
			givenNamespace: "remove-finalizer",
			assert: func(previousResource, result client.Object) {
				ts.Assert().Empty(result.GetFinalizers())
				ts.Assert().NotEqual(previousResource.GetResourceVersion(), result.GetResourceVersion(), "resource version should change")
			},
		},
		"GivenResourceWithoutFinalizer_WhenDeletingFinalizer_ThenExpectResourceUnchanged": {
			prepare: func(resource client.Object) {
				ts.EnsureNS("remove-finalizer")
				ts.EnsureResources(resource)
			},

			givenName:      "no-finalizer",
			givenNamespace: "remove-finalizer",
			assert: func(previousResource, result client.Object) {
				ts.Assert().Empty(result.GetFinalizers())
				ts.Assert().Equal(previousResource.GetResourceVersion(), result.GetResourceVersion(), "resource version should be equal")
			},
		},
	}
	for name, tc := range tests {
		ts.Run(name, func() {
			type resourceKey struct{}
			// Arrange
			resource := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{Name: tc.givenName, Namespace: tc.givenNamespace},
			}
			pipeline.StoreInContext(ts.Context, resourceKey{}, resource)
			tc.prepare(resource)
			previousVersion := resource.DeepCopy()

			// Act
			err := RemoveFinalizerFn(resourceKey{}, "domain.io/finalizer")(ts.Context)
			ts.Require().NoError(err)

			// Assert
			result := &corev1.ConfigMap{}
			ts.FetchResource(client.ObjectKeyFromObject(resource), result)
			tc.assert(previousVersion, result)
		})
	}
}
