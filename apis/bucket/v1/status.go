package v1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// BucketStatus represents the observed state of a Bucket.
type BucketStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty"`
	// BucketName is the name of the actual bucket.
	BucketName string `json:"bucketName,omitempty"`
}

// GetConditions implements conditions.ObjectWithConditions.
func (in *Bucket) GetConditions() []metav1.Condition {
	return in.Status.Conditions
}

// SetConditions implements conditions.ObjectWithConditions.
func (in *Bucket) SetConditions(v []metav1.Condition) {
	in.Status.Conditions = v
}
