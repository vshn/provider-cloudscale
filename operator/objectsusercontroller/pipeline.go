package objectsusercontroller

import (
	"context"

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
	csClient *cloudscalesdk.Client
}

func (p *ObjectsUserPipeline) Disconnect(ctx context.Context) error {
	return nil
}

type pipelineContext struct {
	context.Context
	user              *cloudscalev1.ObjectsUser
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

func hasSecretRef(ctx *pipelineContext) bool {
	return ctx.user.Spec.WriteConnectionSecretToReference != nil
}

func toConnectionDetails(csUser *cloudscalesdk.ObjectsUser) managed.ConnectionDetails {
	if csUser == nil {
		return map[string][]byte{}
	}

	if csUser.Keys == nil {
		return map[string][]byte{}
	}

	if len(csUser.Keys) == 0 {
		return map[string][]byte{}
	}

	if csUser.Keys[0] == nil {
		return map[string][]byte{}
	}

	accessKey, exists := csUser.Keys[0]["access_key"]
	if !exists {
		accessKey = ""
	}

	accessSecret, exists := csUser.Keys[0]["secret_key"]
	if !exists {
		accessSecret = ""
	}

	return map[string][]byte{
		cloudscalev1.AccessKeyIDName:     []byte(accessKey),
		cloudscalev1.SecretAccessKeyName: []byte(accessSecret),
	}
}

func fromManaged(mg resource.Managed) *cloudscalev1.ObjectsUser {
	return mg.(*cloudscalev1.ObjectsUser)
}
