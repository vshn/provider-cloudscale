package v1

import (
	"reflect"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// BucketSpec defines the desired state of a Bucket.
type BucketSpec struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*`

	// CredentialsSecretRef contains the name of the Secret where the credentials of the S3 user are stored.
	// Must be a name that Kubernetes accepts as Secret name (lowercase RFC 1123 subdomain).
	// The secret must contain the keys `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY`.
	CredentialsSecretRef string `json:"credentialsSecretRef"`

	// +kubebuilder:validation:Required

	// EndpointURL is the URL where to create the bucket.
	// If the scheme is omitted (`http/s`), HTTPS is assumed.
	EndpointURL string `json:"endpointURL"`
	// BucketName is the name of the bucket to create.
	// Defaults to `metadata.name` if unset.
	// Cannot be changed after bucket is created.
	// Name must be acceptable by the S3 protocol, which follows RFC 1123.
	// Be aware that S3 providers may require a unique across the platform or region.
	BucketName string `json:"bucketName,omitempty"`

	// +kubebuilder:validation:Required

	// Region is the name of the region where the bucket shall be created.
	// The region must be available in the S3 endpoint.
	Region string `json:"region"`
}

// +kubebuilder:object:root=true
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories={appcat,s3}

// Bucket is the API for creating S3 buckets.
type Bucket struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BucketSpec   `json:"spec"`
	Status BucketStatus `json:"status,omitempty"`
}

// GetBucketName returns the spec.bucketName if given, otherwise defaults to metadata.name.
func (in *Bucket) GetBucketName() string {
	if in.Spec.BucketName == "" {
		return in.Name
	}
	return in.Spec.BucketName
}

// +kubebuilder:object:root=true

// BucketList contains a list of Bucket
type BucketList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Bucket `json:"items"`
}

// Bucket type metadata.
var (
	BucketKind             = reflect.TypeOf(Bucket{}).Name()
	BucketGroupKind        = schema.GroupKind{Group: Group, Kind: BucketKind}.String()
	BucketKindAPIVersion   = BucketKind + "." + SchemeGroupVersion.String()
	BucketGroupVersionKind = SchemeGroupVersion.WithKind(BucketKind)
)

func init() {
	SchemeBuilder.Register(&Bucket{}, &BucketList{})
}
