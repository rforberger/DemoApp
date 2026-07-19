//go:build testutils
// +build testutils

package testutils

import (
	gomega "github.com/onsi/gomega"
    types "github.com/onsi/gomega/types"

    "sigs.k8s.io/kustomize/api/resource"
    "sigs.k8s.io/controller-runtime/pkg/client"
    "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
    "k8s.io/apimachinery/pkg/runtime/serializer/yaml"
)

// Global helper to use Expect inside this package
var Expect gomega.Gomega

// InitGomega binds the external package to the active Ginkgo failure handler
func InitGomega(fail types.GomegaFailHandler) {
	gomega.RegisterFailHandler(fail)
	////Expect = gomega.NewWithT(nil).Expect // Alternative: use gomega.Expect directly after registration
}

// Example helper function using Expect
func VerifyCustomResource() {
	// You can now use the local Expect or standard gomega.Expect
	gomega.Expect(1).To(gomega.Equal(1))
}

// FUNCTION ready
// Create Kubernetes client.Client.Object's from a Kustomize *resource.Resource list
func ResourcesToObjects(resources []*resource.Resource) []client.Object {
	var objects []client.Object
	for _, r := range resources {
		yml, err := r.AsYAML()
		gomega.Expect(err).NotTo(gomega.HaveOccurred(), "🔴 Error converting to YAML with error %s", err)

		// decode the resources into Unstructured
		obj := &unstructured.Unstructured{}
		dec := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)
		_, _, err = dec.Decode(yml, nil, obj)
		gomega.Expect(err).NotTo(gomega.HaveOccurred(), "🔴 Error decoding YAML with error %s", err) // Not perfect to write yml

		// Direct assignment works because *unstructured.Unstructured implements client.Object
		// obj is your Kubernetes object
		var cliObj client.Object = obj

		// 3. Append directly to the slice
		objects = append(objects, cliObj)
	}
	return objects
}