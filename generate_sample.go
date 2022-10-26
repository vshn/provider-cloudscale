//go:build generate

// Clean samples dir
//go:generate rm -rf ./samples/*

// Generate sample files
//go:generate go run generate_sample.go ./samples

package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/vshn/provider-cloudscale/apis"
	cloudscalev1 "github.com/vshn/provider-cloudscale/apis/cloudscale/v1"
	providerv1 "github.com/vshn/provider-cloudscale/apis/provider/v1"
	admissionv1 "k8s.io/api/admission/v1"
	authv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	serializerjson "k8s.io/apimachinery/pkg/runtime/serializer/json"
)

var scheme = runtime.NewScheme()

func main() {
	failIfError(apis.AddToScheme(scheme))
	generateCloudscaleObjectsUserSample()
	generateBucketSample()
	generateProviderConfigSample()
	generateBucketAdmissionRequest()
}

func generateCloudscaleObjectsUserSample() {
	spec := newObjectsUserSample()
	serialize(spec, true)
}

func generateBucketSample() {
	spec := newBucketSample()
	serialize(spec, true)
}

func generateProviderConfigSample() {
	spec := newProviderConfigSample()
	serialize(spec, true)
}

func newObjectsUserSample() *cloudscalev1.ObjectsUser {
	return &cloudscalev1.ObjectsUser{
		TypeMeta: metav1.TypeMeta{
			APIVersion: cloudscalev1.ObjectsUserGroupVersionKind.GroupVersion().String(),
			Kind:       cloudscalev1.ObjectsUserKind,
		},
		ObjectMeta: metav1.ObjectMeta{Name: "my-cloudscale-user"},
		Spec: cloudscalev1.ObjectsUserSpec{
			ResourceSpec: xpv1.ResourceSpec{
				ProviderConfigReference: &xpv1.Reference{Name: "provider-config"},
				WriteConnectionSecretToReference: &xpv1.SecretReference{
					Name:      "my-cloudscale-user-credentials",
					Namespace: "default",
				},
			},
			ForProvider: cloudscalev1.ObjectsUserParameters{
				Tags: map[string]string{
					"key": "value",
				},
			},
		},
		Status: cloudscalev1.ObjectsUserStatus{},
	}
}

func newBucketSample() *cloudscalev1.Bucket {
	return &cloudscalev1.Bucket{
		TypeMeta: metav1.TypeMeta{
			APIVersion: cloudscalev1.BucketGroupVersionKind.GroupVersion().String(),
			Kind:       cloudscalev1.BucketKind,
		},
		ObjectMeta: metav1.ObjectMeta{Name: "bucket"},
		Spec: cloudscalev1.BucketSpec{
			ForProvider: cloudscalev1.BucketParameters{
				CredentialsSecretRef: corev1.SecretReference{
					Name:      "my-cloudscale-user-credentials",
					Namespace: "default",
				},
				BucketName:           "my-provider-test-bucket",
				Region:               "rma",
				BucketDeletionPolicy: cloudscalev1.DeleteAll,
			},
		},
	}
}

func newProviderConfigSample() *providerv1.ProviderConfig {
	return &providerv1.ProviderConfig{
		TypeMeta: metav1.TypeMeta{
			APIVersion: providerv1.ProviderConfigGroupVersionKind.GroupVersion().String(),
			Kind:       providerv1.ProviderConfigKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "provider-config"},
		Spec: providerv1.ProviderConfigSpec{
			Credentials: providerv1.ProviderCredentials{
				Source: xpv1.CredentialsSourceInjectedIdentity,
				APITokenSecretRef: corev1.SecretReference{
					Name:      "api-token",
					Namespace: "crossplane-system",
				},
			},
		},
	}
}

// generateBucketAdmissionRequest generates an update request that will fail.
func generateBucketAdmissionRequest() {
	oldSpec := newBucketSample()
	newSpec := newBucketSample()
	newSpec.Spec.ForProvider.BucketName = "another"
	oldSpec.Status.AtProvider.BucketName = oldSpec.Spec.ForProvider.BucketName

	gvk := metav1.GroupVersionKind{Group: cloudscalev1.Group, Version: cloudscalev1.Version, Kind: cloudscalev1.BucketKind}
	gvr := metav1.GroupVersionResource{Group: cloudscalev1.Group, Version: cloudscalev1.Version, Resource: cloudscalev1.BucketKind}
	admission := &admissionv1.AdmissionReview{
		TypeMeta: metav1.TypeMeta{APIVersion: "admission.k8s.io/v1", Kind: "AdmissionReview"},
		Request: &admissionv1.AdmissionRequest{
			Object:          runtime.RawExtension{Object: newSpec},
			OldObject:       runtime.RawExtension{Object: oldSpec},
			Kind:            gvk,
			Resource:        gvr,
			RequestKind:     &gvk,
			RequestResource: &gvr,
			Name:            oldSpec.Name,
			Operation:       admissionv1.Update,
			UserInfo: authv1.UserInfo{
				Username: "admin",
				Groups:   []string{"system:authenticated"},
			},
		},
	}
	serialize(admission, false)
}

func failIfError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func serialize(object runtime.Object, useYaml bool) {
	gvk := object.GetObjectKind().GroupVersionKind()
	fileExt := "json"
	if useYaml {
		fileExt = "yaml"
	}
	fileName := fmt.Sprintf("%s_%s.%s", strings.ToLower(gvk.Group), strings.ToLower(gvk.Kind), fileExt)
	f := prepareFile(fileName)
	err := serializerjson.NewSerializerWithOptions(serializerjson.DefaultMetaFactory, scheme, scheme, serializerjson.SerializerOptions{Yaml: useYaml, Pretty: true}).Encode(object, f)
	failIfError(err)
}

func prepareFile(file string) io.Writer {
	dir := os.Args[1]
	err := os.MkdirAll(os.Args[1], 0775)
	failIfError(err)
	f, err := os.Create(filepath.Join(dir, file))
	failIfError(err)
	return f
}
