//go:build integration

package cloudscale

import (
	"context"
	"testing"

	pipeline "github.com/ccremer/go-command-pipeline"
	cloudscalesdk "github.com/cloudscale-ch/cloudscale-go-sdk/v2"
	"github.com/stretchr/testify/suite"
	cloudscalev1 "github.com/vshn/provider-cloudscale/apis/cloudscale/v1"
	"github.com/vshn/provider-cloudscale/operator/operatortest"
	"github.com/vshn/provider-cloudscale/operator/steps"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

type ProvisionPipelineSuite struct {
	operatortest.Suite
}

func TestProvisionPipelineSuite(t *testing.T) {
	suite.Run(t, new(ProvisionPipelineSuite))
}

func (ts *ProvisionPipelineSuite) BeforeTest(suiteName, testName string) {
	ts.Context = pipeline.MutableContext(context.Background())
	steps.SetClientInContext(ts.Context, ts.Client)
}

func (ts *ProvisionPipelineSuite) Test_EnsureCredentialSecretFn() {
	// Arrange
	user := &cloudscalev1.ObjectsUser{
		ObjectMeta: metav1.ObjectMeta{Name: "user", UID: "uid"},
		Spec:       cloudscalev1.ObjectsUserSpec{ForProvider: cloudscalev1.ObjectsUserParameters{SecretRef: corev1.SecretReference{Name: "secret", Namespace: "namespace"}}}}
	pipeline.StoreInContext(ts.Context, ObjectsUserKey{}, user)

	csUser := &cloudscalesdk.ObjectsUser{
		Keys: []map[string]string{{"access_key": "access", "secret_key": "secret"}},
	}
	pipeline.StoreInContext(ts.Context, CloudscaleUserKey{}, csUser)

	ts.EnsureNS(user.Namespace)

	// Act
	err := ensureCredentialSecret(ts.Context)
	ts.Require().NoError(err)

	// Assert
	result := &corev1.Secret{}
	ts.FetchResource(types.NamespacedName{Namespace: "namespace", Name: "secret"}, result)
	ts.Require().Len(result.Data, 2, "amount of keys")
	ts.Assert().Equal("access", string(result.Data["AWS_ACCESS_KEY_ID"]), "access key value")
	ts.Assert().Equal("secret", string(result.Data["AWS_SECRET_ACCESS_KEY"]), "secret key value")
	ts.Assert().Equal(types.UID("uid"), result.OwnerReferences[0].UID, "owner reference set")
	ts.Assert().Contains(result.Finalizers, userFinalizer, "finalizer present")
	ts.Assert().NotNil(pipeline.MustLoadFromContext(ts.Context, UserCredentialSecretKey{}), "secret stored in context")
}
