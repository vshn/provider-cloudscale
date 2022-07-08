//go:build integration

package cloudscale

import (
	"context"
	"fmt"
	pipeline "github.com/ccremer/go-command-pipeline"
	"github.com/stretchr/testify/suite"
	cloudscalev1 "github.com/vshn/appcat-service-s3/apis/cloudscale/v1"
	"github.com/vshn/appcat-service-s3/apis/conditions"
	"github.com/vshn/appcat-service-s3/operator/operatortest"
	"github.com/vshn/appcat-service-s3/operator/steps"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"testing"
)

type ControllerSuite struct {
	operatortest.Suite
}

func TestControllerSuite(t *testing.T) {
	suite.Run(t, new(ControllerSuite))
}

func (ts *ControllerSuite) BeforeTest(suiteName, testName string) {
	ts.Context = pipeline.MutableContext(context.Background())
	steps.SetClientInContext(ts.Context, ts.Client)
}

func (ts *ControllerSuite) Test_ErrorHandler() {
	// Arrange
	user := &cloudscalev1.ObjectsUser{
		ObjectMeta: metav1.ObjectMeta{Name: "user", Namespace: "namespace"},
		Spec:       cloudscalev1.ObjectsUserSpec{SecretRef: "irrelevant-but-required"}}
	pipeline.StoreInContext(ts.Context, ObjectsUserKey{}, user)
	ts.EnsureNS(user.Namespace)
	ts.EnsureResources(user)

	mgr, err := ctrl.NewManager(ts.Config, ctrl.Options{Scheme: ts.Scheme})
	ts.Require().NoError(err)

	recorder := mgr.GetEventRecorderFor("controller")
	steps.SetEventRecorderInContext(ts.Context, recorder)

	// Act
	result := pipeline.NewPipeline().
		WithSteps(pipeline.NewStepFromFunc("create error", func(ctx context.Context) error {
			return fmt.Errorf("error")
		})).
		WithFinalizer(errorHandler()).
		RunWithContext(ts.Context)

	// Assert
	updatedUser := &cloudscalev1.ObjectsUser{}
	ts.FetchResource(client.ObjectKeyFromObject(user), updatedUser)
	ts.Assert().EqualError(result.Err(), `step "create error" failed: error`, "error returned")
	ts.Require().Len(user.Status.Conditions, 2, "amount of conditions")

	readyCondition := user.Status.Conditions[0]
	ts.Assert().Equal(metav1.ConditionFalse, readyCondition.Status)
	ts.Assert().Equal(conditions.TypeReady, readyCondition.Type)
	failedCondition := user.Status.Conditions[1]
	ts.Assert().Equal(metav1.ConditionTrue, failedCondition.Status)
	ts.Assert().Equal(conditions.TypeFailed, failedCondition.Type)
	ts.Assert().Equal(`step "create error" failed: error`, failedCondition.Message, "error message in condition")
}
