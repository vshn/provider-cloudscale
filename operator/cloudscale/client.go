package cloudscale

import (
	"context"
	"fmt"
	pipeline "github.com/ccremer/go-command-pipeline"
	cloudscalesdk "github.com/cloudscale-ch/cloudscale-go-sdk/v2"
	cloudscalev1 "github.com/vshn/appcat-service-s3/apis/cloudscale/v1"
	"github.com/vshn/appcat-service-s3/operator/steps"
	"golang.org/x/oauth2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// APIToken is the authentication token to use against cloudscale.ch API
var APIToken string

// CloudscaleClientKey identifies the cloudscale client in the context.
type CloudscaleClientKey struct{}

// CreateCloudscaleClientFn creates a new client using the API token provided.
func CreateCloudscaleClientFn(apiToken string) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		tc := oauth2.NewClient(ctx, oauth2.StaticTokenSource(&oauth2.Token{AccessToken: apiToken}))
		csClient := cloudscalesdk.NewClient(tc)
		pipeline.StoreInContext(ctx, CloudscaleClientKey{}, csClient)
		return nil
	}
}

// CloudscaleUserKey identifies the User object from cloudscale SDK in the context.
type CloudscaleUserKey struct{}

// CreateObjectsUser creates a new objects user in the project associated with the API token.
func CreateObjectsUser(ctx context.Context) error {
	csClient := steps.GetFromContextOrPanic(ctx, CloudscaleClientKey{}).(*cloudscalesdk.Client)
	user := steps.GetFromContextOrPanic(ctx, ObjectsUserKey{}).(*cloudscalev1.ObjectsUser)
	log := controllerruntime.LoggerFrom(ctx)

	displayName := fmt.Sprintf("%s.%s", user.Namespace, user.Name)

	csUser, err := csClient.ObjectsUsers.Create(ctx, &cloudscalesdk.ObjectsUserRequest{
		DisplayName: displayName,
	})
	user.Status.UserID = csUser.ID

	pipeline.StoreInContext(ctx, CloudscaleUserKey{}, csUser)
	return logIfNotError(err, log, 1, "Created objects user in cloudscale")
}

// GetObjectsUser fetches an existing objects user from the project associated with the API token.
func GetObjectsUser(ctx context.Context) error {
	csClient := steps.GetFromContextOrPanic(ctx, CloudscaleClientKey{}).(*cloudscalesdk.Client)
	user := steps.GetFromContextOrPanic(ctx, ObjectsUserKey{}).(*cloudscalev1.ObjectsUser)
	log := controllerruntime.LoggerFrom(ctx)

	csUser, err := csClient.ObjectsUsers.Get(ctx, user.Status.UserID)

	pipeline.StoreInContext(ctx, CloudscaleUserKey{}, csUser)
	return logIfNotError(err, log, 1, "Fetched objects user in cloudscale")
}

// UserCredentialSecretKey identifies the credential Secret in the context.
type UserCredentialSecretKey struct{}

// EnsureCredentialSecret creates the credential secret.
func EnsureCredentialSecret(ctx context.Context) error {
	kube := steps.GetClientFromContext(ctx)
	user := steps.GetFromContextOrPanic(ctx, ObjectsUserKey{}).(*cloudscalev1.ObjectsUser)
	csUser := steps.GetFromContextOrPanic(ctx, CloudscaleUserKey{}).(*cloudscalesdk.ObjectsUser)
	log := controllerruntime.LoggerFrom(ctx)

	secret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: user.Spec.SecretRef, Namespace: user.Namespace}}

	if keyErr := checkUserForKeys(csUser); keyErr != nil {
		return keyErr
	}

	// See https://www.cloudscale.ch/en/api/v1#objects-users

	_, err := controllerruntime.CreateOrUpdate(ctx, kube, secret, func() error {
		secret.Labels = labels.Merge(secret.Labels, getCommonLabels(user.Name))
		if secret.StringData == nil {
			secret.StringData = map[string]string{}
		}
		secret.StringData["AWS_ACCESS_KEY_ID"] = csUser.Keys[0]["access_key"]
		secret.StringData["AWS_SECRET_ACCESS_KEY"] = csUser.Keys[0]["secret_key"]
		controllerutil.AddFinalizer(secret, userFinalizer)
		return controllerutil.SetOwnerReference(user, secret, kube.Scheme())
	})

	pipeline.StoreInContext(ctx, UserCredentialSecretKey{}, secret)
	return logIfNotError(err, log, 1, "Ensured credential secret", "secretName", user.Spec.SecretRef)
}

func checkUserForKeys(user *cloudscalesdk.ObjectsUser) error {
	if len(user.Keys) == 0 {
		return fmt.Errorf("the returned objects user has no key pairs: %q", user.ID)
	}
	if val, exists := user.Keys[0]["secret_key"]; exists && val == "" {
		return fmt.Errorf("the returned objects user %q has no secret_key. Does the API token have enough permissions?", user.ID)
	}
	return nil
}
