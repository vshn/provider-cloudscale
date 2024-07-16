package bucketcontroller

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	cloudscalev1 "github.com/vshn/provider-cloudscale/apis/cloudscale/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestBucketValidator_ValidateUpdate_PreventBucketNameChange(t *testing.T) {
	tests := map[string]struct {
		newBucketName string
		oldBucketName string
		expectedError string
	}{
		"GivenNoNameInStatus_WhenNoNameInSpec_ThenExpectNil": {
			oldBucketName: "",
			newBucketName: "",
		},
		"GivenNoNameInStatus_WhenNameInSpec_ThenExpectNil": {
			oldBucketName: "",
			newBucketName: "my-bucket",
		},
		"GivenNameInStatus_WhenNameInSpecSame_ThenExpectNil": {
			oldBucketName: "my-bucket",
			newBucketName: "my-bucket",
		},
		"GivenNameInStatus_WhenNameInSpecEmpty_ThenExpectNil": {
			oldBucketName: "bucket",
			newBucketName: "", // defaults to metadata.name
		},
		"GivenNameInStatus_WhenNameInSpecDifferent_ThenExpectError": {
			oldBucketName: "my-bucket",
			newBucketName: "different",
			expectedError: `a bucket named "my-bucket" has been created already, you cannot rename it`,
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			oldBucket := &cloudscalev1.Bucket{
				ObjectMeta: metav1.ObjectMeta{Name: "bucket"},
				Spec:       cloudscalev1.BucketSpec{ForProvider: cloudscalev1.BucketParameters{BucketName: tc.oldBucketName}},
				Status:     cloudscalev1.BucketStatus{AtProvider: cloudscalev1.BucketObservation{BucketName: tc.oldBucketName}},
			}
			newBucket := &cloudscalev1.Bucket{
				ObjectMeta: metav1.ObjectMeta{Name: "bucket"},
				Spec:       cloudscalev1.BucketSpec{ForProvider: cloudscalev1.BucketParameters{BucketName: tc.newBucketName}},
			}
			v := &BucketValidator{log: logr.Discard()}
			_, err := v.ValidateUpdate(context.TODO(), oldBucket, newBucket)
			if tc.expectedError != "" {
				assert.EqualError(t, err, tc.expectedError)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestBucketValidator_ValidateUpdate_PreventRegionChange(t *testing.T) {
	tests := map[string]struct {
		newRegion     string
		oldRegion     string
		expectedError string
	}{
		"GivenRegionUnchanged_ThenExpectNil": {
			oldRegion: "region",
			newRegion: "region",
		},
		"GivenRegionChanged_ThenExpectError": {
			oldRegion:     "region",
			newRegion:     "different",
			expectedError: `a bucket named "bucket" has been created already, you cannot change the region`,
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			oldBucket := &cloudscalev1.Bucket{
				ObjectMeta: metav1.ObjectMeta{Name: "bucket"},
				Spec:       cloudscalev1.BucketSpec{ForProvider: cloudscalev1.BucketParameters{Region: tc.oldRegion}},
				Status:     cloudscalev1.BucketStatus{AtProvider: cloudscalev1.BucketObservation{BucketName: "bucket"}},
			}
			newBucket := &cloudscalev1.Bucket{
				ObjectMeta: metav1.ObjectMeta{Name: "bucket"},
				Spec:       cloudscalev1.BucketSpec{ForProvider: cloudscalev1.BucketParameters{Region: tc.newRegion}},
				Status:     cloudscalev1.BucketStatus{AtProvider: cloudscalev1.BucketObservation{BucketName: "bucket"}},
			}
			v := &BucketValidator{log: logr.Discard()}
			_, err := v.ValidateUpdate(context.TODO(), oldBucket, newBucket)
			if tc.expectedError != "" {
				assert.EqualError(t, err, tc.expectedError)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
