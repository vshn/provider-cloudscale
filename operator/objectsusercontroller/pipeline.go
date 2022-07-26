package objectsusercontroller

import (
	"context"

	pipeline "github.com/ccremer/go-command-pipeline"
	cloudscalesdk "github.com/cloudscale-ch/cloudscale-go-sdk/v2"
	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	cloudscalev1 "github.com/vshn/provider-cloudscale/apis/cloudscale/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// UserIDAnnotationKey is the annotation key where the ObjectsUser ID is stored.
	UserIDAnnotationKey = "cloudscale.crossplane.io/user-id"
)

// ObjectsUserPipeline provisions ObjectsUsers on cloudscale.ch
type ObjectsUserPipeline struct {
	kube     client.Client
	recorder event.Recorder

	csClient          *cloudscalesdk.Client
	csUser            *cloudscalesdk.ObjectsUser
	credentialsSecret *corev1.Secret
}

// NewPipeline returns a new instance of ObjectsUserPipeline.
func NewPipeline(client client.Client, recorder event.Recorder, csClient *cloudscalesdk.Client) *ObjectsUserPipeline {
	return &ObjectsUserPipeline{
		kube:     client,
		recorder: recorder,
		csClient: csClient,
	}
}

func hasSecretRef(user *cloudscalev1.ObjectsUser) pipeline.Predicate {
	return func(ctx context.Context) bool {
		return user.Spec.WriteConnectionSecretToReference != nil
	}
}

func toConnectionDetails(csUser *cloudscalesdk.ObjectsUser) managed.ConnectionDetails {
	return map[string][]byte{
		cloudscalev1.AccessKeyIDName:     []byte(csUser.Keys[0]["access_key"]),
		cloudscalev1.SecretAccessKeyName: []byte(csUser.Keys[0]["secret_key"]),
	}
}

func fromManaged(mg resource.Managed) *cloudscalev1.ObjectsUser {
	return mg.(*cloudscalev1.ObjectsUser)
}
