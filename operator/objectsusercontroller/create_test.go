package objectsusercontroller

import (
	"context"
	"testing"

	pipeline "github.com/ccremer/go-command-pipeline"
	cloudscalesdk "github.com/cloudscale-ch/cloudscale-go-sdk/v2"
	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/stretchr/testify/suite"
	cloudscalev1 "github.com/vshn/provider-cloudscale/apis/cloudscale/v1"
	"github.com/vshn/provider-cloudscale/operator/operatortest"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

type ObjectsUserPipelineSuite struct {
	operatortest.Suite
}

func TestProvisionPipelineSuite(t *testing.T) {
	suite.Run(t, new(ObjectsUserPipelineSuite))
}

func (ts *ObjectsUserPipelineSuite) BeforeTest(suiteName, testName string) {
	ts.Context = pipeline.MutableContext(context.Background())
}

func (ts *ObjectsUserPipelineSuite) Test_ensureCredentialSecretFn() {
	// Arrange
	user := &cloudscalev1.ObjectsUser{
		ObjectMeta: metav1.ObjectMeta{Name: "user", UID: "uid"},
		Spec: cloudscalev1.ObjectsUserSpec{ResourceSpec: xpv1.ResourceSpec{
			WriteConnectionSecretToReference: &xpv1.SecretReference{
				Name: "secret", Namespace: "ensure-credentials",
			}},
		},
	}

	csUser := &cloudscalesdk.ObjectsUser{
		Keys: []map[string]string{{"access_key": "access", "secret_key": "secret"}},
	}

	ts.EnsureNS(user.Spec.WriteConnectionSecretToReference.Namespace)

	p := ObjectsUserPipeline{
		kube: ts.Client,
	}

	// Act
	ctx := &pipelineContext{Context: ts.Context, user: user, csUser: csUser}
	err := p.ensureCredentialsSecret(ctx)
	ts.Require().NoError(err)

	// Assert
	result := &corev1.Secret{}
	ts.FetchResource(types.NamespacedName{Namespace: "ensure-credentials", Name: "secret"}, result)
	ts.Require().Len(result.Data, 2, "amount of keys")
	ts.Assert().Equal("access", string(result.Data["AWS_ACCESS_KEY_ID"]), "access key value")
	ts.Assert().Equal("secret", string(result.Data["AWS_SECRET_ACCESS_KEY"]), "secret key value")
	ts.Assert().Equal(types.UID("uid"), result.OwnerReferences[0].UID, "owner reference set")
}
