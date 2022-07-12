//go:build generate

// Clean samples dir
//go:generate rm -rf package/samples/*

// Generate sample files
//go:generate go run generate_sample.go package/samples

package main

import (
	"fmt"
	"github.com/vshn/appcat-service-s3/apis"
	bucketv1 "github.com/vshn/appcat-service-s3/apis/bucket/v1"
	cloudscalev1 "github.com/vshn/appcat-service-s3/apis/cloudscale/v1"
	"io"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	serializerjson "k8s.io/apimachinery/pkg/runtime/serializer/json"
	"log"
	"os"
	"path/filepath"
	"strings"
)

var scheme = runtime.NewScheme()

func main() {
	failIfError(apis.AddToScheme(scheme))
	generateCloudscaleObjectsUserSample()
	generateBucketSample()
}

func generateCloudscaleObjectsUserSample() {
	spec := newObjectsUserSample()
	serialize(spec, true)
}

func generateBucketSample() {
	spec := newBucketSample()
	serialize(spec, true)
}

func newObjectsUserSample() *cloudscalev1.ObjectsUser {
	return &cloudscalev1.ObjectsUser{
		TypeMeta: metav1.TypeMeta{
			APIVersion: cloudscalev1.ObjectsUserGroupVersionKind.GroupVersion().String(),
			Kind:       cloudscalev1.ObjectsUserKind,
		},
		ObjectMeta: metav1.ObjectMeta{Name: "my-cloudscale-user", Namespace: "default", Generation: 1},
		Spec: cloudscalev1.ObjectsUserSpec{
			SecretRef: "my-cloudscale-user-credentials",
		},
		Status: cloudscalev1.ObjectsUserStatus{},
	}
}

func newBucketSample() *bucketv1.Bucket {
	return &bucketv1.Bucket{
		TypeMeta: metav1.TypeMeta{
			APIVersion: bucketv1.BucketGroupVersionKind.GroupVersion().String(),
			Kind:       bucketv1.BucketKind,
		},
		ObjectMeta: metav1.ObjectMeta{Name: "bucket", Namespace: "default"},
		Spec: bucketv1.BucketSpec{
			CredentialsSecretRef: "my-cloudscale-user-credentials",
			EndpointURL:          "objects.rma.cloudscale.ch",
			BucketName:           "my-appcat-test-bucket",
			Region:               "rma",
		},
	}
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
