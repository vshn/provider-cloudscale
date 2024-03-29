package bucketcontroller

import (
	"context"
	"net/http"
	"testing"

	"github.com/crossplane/crossplane-runtime/pkg/errors"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/go-logr/logr"
	"github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	cloudscalev1 "github.com/vshn/provider-cloudscale/apis/cloudscale/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestProvisioningPipeline_Observe(t *testing.T) {
	tests := map[string]struct {
		givenBucket  *cloudscalev1.Bucket
		bucketExists bool
		returnError  error

		expectedError             string
		expectedResult            managed.ExternalObservation
		expectedBucketObservation cloudscalev1.BucketObservation
	}{
		"NewBucketDoesntYetExistOnCloudscale": {
			givenBucket: &cloudscalev1.Bucket{Spec: cloudscalev1.BucketSpec{ForProvider: cloudscalev1.BucketParameters{
				BucketName: "my-bucket"}},
			},
			expectedResult: managed.ExternalObservation{},
		},
		"BucketExistsAndAccessibleWithOurCredentials": {
			givenBucket: &cloudscalev1.Bucket{
				ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{
					lockAnnotation: "claimed",
				}},
				Spec: cloudscalev1.BucketSpec{ForProvider: cloudscalev1.BucketParameters{
					BucketName: "my-bucket"}},
			},
			bucketExists:              true,
			expectedResult:            managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: true},
			expectedBucketObservation: cloudscalev1.BucketObservation{BucketName: "my-bucket"},
		},
		"NewBucketObservationThrowsGenericError": {
			givenBucket: &cloudscalev1.Bucket{Spec: cloudscalev1.BucketSpec{ForProvider: cloudscalev1.BucketParameters{
				BucketName: "my-bucket"}},
			},
			returnError:    errors.New("error"),
			expectedResult: managed.ExternalObservation{},
			expectedError:  "cannot determine whether bucket exists: error",
		},
		"BucketAlreadyExistsOnCloudscale_WithoutAccess": {
			givenBucket: &cloudscalev1.Bucket{Spec: cloudscalev1.BucketSpec{ForProvider: cloudscalev1.BucketParameters{
				BucketName: "my-bucket"}},
			},
			returnError:    minio.ErrorResponse{StatusCode: http.StatusForbidden, Message: "Access Denied"},
			expectedResult: managed.ExternalObservation{},
			expectedError:  "wrong credentials or bucket exists already, try changing bucket name: Access Denied",
		},
		"BucketAlreadyExistsOnCloudscale_WithAccess_PreventAdoption": {
			// this is a case where we should avoid adopting an existing bucket even if we have access.
			// Otherwise, there could be multiple K8s resources that manage the same bucket.
			givenBucket: &cloudscalev1.Bucket{
				Spec: cloudscalev1.BucketSpec{ForProvider: cloudscalev1.BucketParameters{
					BucketName: "my-bucket"}},
				// no bucket name in status here.
			},
			bucketExists:   true,
			expectedResult: managed.ExternalObservation{},
			expectedError:  "bucket exists already, try changing bucket name: my-bucket",
		},
		"BucketAlreadyExistsOnCloudscale_InAnotherZone": {
			givenBucket: &cloudscalev1.Bucket{
				Spec: cloudscalev1.BucketSpec{ForProvider: cloudscalev1.BucketParameters{
					BucketName: "my-bucket"}},
			},
			returnError:    minio.ErrorResponse{StatusCode: http.StatusMovedPermanently, Message: "301 Moved Permanently"},
			expectedResult: managed.ExternalObservation{},
			expectedError:  "mismatching endpointURL and region, or bucket exists already in a different region, try changing bucket name: 301 Moved Permanently",
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			currFn := bucketExistsFn
			defer func() {
				bucketExistsFn = currFn
			}()
			bucketExistsFn = func(ctx context.Context, mc *minio.Client, bucketName string) (bool, error) {
				return tc.bucketExists, tc.returnError
			}
			p := ProvisioningPipeline{}
			result, err := p.Observe(logr.NewContext(context.Background(), logr.Discard()), tc.givenBucket)
			if tc.expectedError != "" {
				assert.EqualError(t, err, tc.expectedError)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.expectedResult, result)
			assert.Equal(t, tc.expectedBucketObservation, tc.givenBucket.Status.AtProvider)
		})
	}
}
