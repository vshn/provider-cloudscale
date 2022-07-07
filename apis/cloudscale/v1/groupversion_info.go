// +kubebuilder:object:generate=true
// +groupName=cloudscale.s3.appcat.vshn.io
// +versionName=v1

// Package v1 contains the v1 group cloudscale.s3.appcat.vshn.io resources of the S3 provider.
package v1

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

// Package type metadata.
const (
	Group   = "cloudscale.s3.appcat.vshn.io"
	Version = "v1"
)

var (
	// SchemeGroupVersion is group version used to register these objects
	SchemeGroupVersion = schema.GroupVersion{Group: Group, Version: Version}

	// SchemeBuilder is used to add go types to the GroupVersionKind scheme
	SchemeBuilder = &scheme.Builder{GroupVersion: SchemeGroupVersion}
)
