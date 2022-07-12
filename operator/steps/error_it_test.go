package steps

import (
	"context"
	"fmt"
	"testing"

	pipeline "github.com/ccremer/go-command-pipeline"
	"github.com/stretchr/testify/suite"
	bucketv1 "github.com/vshn/appcat-service-s3/apis/bucket/v1"
	"github.com/vshn/appcat-service-s3/apis/conditions"
	"github.com/vshn/appcat-service-s3/operator/operatortest"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ErrorSuite struct {
	operatortest.Suite
}

func TestErrorSuite(t *testing.T) {
	suite.Run(t, new(ErrorSuite))
}

func (ts *ErrorSuite) BeforeTest(suiteName, testName string) {
	ts.Context = pipeline.MutableContext(context.Background())
	ts.RegisterScheme(bucketv1.SchemeBuilder.AddToScheme)
	SetClientInContext(ts.Context, ts.Client)
}

func (ts *ErrorSuite) Test_ErrorHandler() {
	// Arrange
	type objKey struct{}

	obj := &bucketv1.Bucket{ // need an object that implements conditions.ObjectWithConditions.
		ObjectMeta: metav1.ObjectMeta{Name: "obj", Namespace: "namespace"},
		Spec:       bucketv1.BucketSpec{CredentialsSecretRef: "irrelevant-but-required"}}
	pipeline.StoreInContext(ts.Context, objKey{}, obj)
	ts.EnsureNS(obj.Namespace)
	ts.EnsureResources(obj)

	mgr, err := ctrl.NewManager(ts.Config, ctrl.Options{Scheme: ts.Scheme})
	ts.Require().NoError(err)

	recorder := mgr.GetEventRecorderFor("controller")
	SetEventRecorderInContext(ts.Context, recorder)

	// Act
	result := pipeline.NewPipeline().
		WithSteps(pipeline.NewStepFromFunc("create error", func(ctx context.Context) error {
			return fmt.Errorf("error")
		})).
		WithFinalizer(ErrorHandlerFn(objKey{}, conditions.ReasonProvisioningFailed)).
		RunWithContext(ts.Context)

	// Assert
	updatedObj := &bucketv1.Bucket{}
	ts.FetchResource(client.ObjectKeyFromObject(obj), updatedObj)
	ts.Assert().EqualError(result.Err(), `step "create error" failed: error`, "error returned")
	ts.Require().Len(obj.Status.Conditions, 2, "amount of conditions")

	readyCondition := obj.Status.Conditions[0]
	ts.Assert().Equal(metav1.ConditionFalse, readyCondition.Status)
	ts.Assert().Equal(conditions.TypeReady, readyCondition.Type)
	failedCondition := obj.Status.Conditions[1]
	ts.Assert().Equal(metav1.ConditionTrue, failedCondition.Status)
	ts.Assert().Equal(conditions.TypeFailed, failedCondition.Type)
	ts.Assert().Equal(conditions.ReasonProvisioningFailed, failedCondition.Reason)
	ts.Assert().Equal(`step "create error" failed: error`, failedCondition.Message, "error message in condition")
}
