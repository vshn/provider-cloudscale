//go:build integration

package cloudscale

import (
	"context"
	pipeline "github.com/ccremer/go-command-pipeline"
	cloudscalesdk "github.com/cloudscale-ch/cloudscale-go-sdk/v2"
	"github.com/stretchr/testify/suite"
	cloudscalev1 "github.com/vshn/appcat-service-s3/apis/cloudscale/v1"
	"github.com/vshn/appcat-service-s3/operator/operatortest"
	"github.com/vshn/appcat-service-s3/operator/steps"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"testing"
)

type CloudscaleClientSuite struct {
	operatortest.Suite
}

func TestFinalizerSuite(t *testing.T) {
	suite.Run(t, new(CloudscaleClientSuite))
}

func (ts *CloudscaleClientSuite) BeforeTest(suiteName, testName string) {
	ts.Context = pipeline.MutableContext(context.Background())
	steps.SetClientInContext(ts.Context, ts.Client)
}

func (ts *CloudscaleClientSuite) Test_EnsureCredentialSecretFn() {
	// Arrange
	user := &cloudscalev1.ObjectsUser{
		ObjectMeta: metav1.ObjectMeta{Name: "user", Namespace: "namespace", UID: "uid"},
		Spec:       cloudscalev1.ObjectsUserSpec{SecretRef: "secret"}}
	pipeline.StoreInContext(ts.Context, ObjectsUserKey{}, user)

	csUser := &cloudscalesdk.ObjectsUser{
		Keys: []map[string]string{{"access_key": "access", "secret_key": "secret"}},
	}
	pipeline.StoreInContext(ts.Context, CloudscaleUserKey{}, csUser)

	ts.EnsureNS(user.Namespace)

	// Act
	err := EnsureCredentialSecret(ts.Context)
	ts.Require().NoError(err)

	// Assert
	result := &corev1.Secret{}
	ts.FetchResource(types.NamespacedName{Namespace: user.Namespace, Name: "secret"}, result)
	ts.Require().Len(result.Data, 2, "amount of keys")
	ts.Assert().Equal("access", string(result.Data["AWS_ACCESS_KEY_ID"]), "access key value")
	ts.Assert().Equal("secret", string(result.Data["AWS_SECRET_ACCESS_KEY"]), "secret key value")
	ts.Assert().Equal(types.UID("uid"), result.OwnerReferences[0].UID, "owner reference set")
	ts.Assert().Contains(result.Finalizers, userFinalizer, "finalizer present")
	ts.Assert().NotNil(pipeline.MustLoadFromContext(ts.Context, UserCredentialSecretKey{}), "secret stored in context")
}
