package cloudscale

import (
	"context"
	"fmt"

	pipeline "github.com/ccremer/go-command-pipeline"
	cloudscalesdk "github.com/cloudscale-ch/cloudscale-go-sdk/v2"
	cloudscalev1 "github.com/vshn/provider-cloudscale/apis/cloudscale/v1"
	"github.com/vshn/provider-cloudscale/operator/steps"
	"golang.org/x/oauth2"
	controllerruntime "sigs.k8s.io/controller-runtime"
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
	user.Status.AtProvider.UserID = csUser.ID

	pipeline.StoreInContext(ctx, CloudscaleUserKey{}, csUser)
	return logIfNotError(err, log, 1, "Created objects user in cloudscale", "userID", csUser.ID)
}

// GetObjectsUser fetches an existing objects user from the project associated with the API token.
func GetObjectsUser(ctx context.Context) error {
	csClient := steps.GetFromContextOrPanic(ctx, CloudscaleClientKey{}).(*cloudscalesdk.Client)
	user := steps.GetFromContextOrPanic(ctx, ObjectsUserKey{}).(*cloudscalev1.ObjectsUser)
	log := controllerruntime.LoggerFrom(ctx)

	csUser, err := csClient.ObjectsUsers.Get(ctx, user.Status.AtProvider.UserID)

	pipeline.StoreInContext(ctx, CloudscaleUserKey{}, csUser)
	return logIfNotError(err, log, 1, "Fetched objects user in cloudscale")
}

// DeleteObjectsUser deletes the objects user from the project associated with the API token.
func DeleteObjectsUser(ctx context.Context) error {
	csClient := steps.GetFromContextOrPanic(ctx, CloudscaleClientKey{}).(*cloudscalesdk.Client)
	user := steps.GetFromContextOrPanic(ctx, ObjectsUserKey{}).(*cloudscalev1.ObjectsUser)
	log := controllerruntime.LoggerFrom(ctx)

	err := csClient.ObjectsUsers.Delete(ctx, user.Status.AtProvider.UserID)
	return logIfNotError(err, log, 1, "Deleted objects user in cloudscale", "userID", user.Status.AtProvider.UserID)
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
