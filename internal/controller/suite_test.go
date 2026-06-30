package controller

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	//corev1 "k8s.io/api/core/v1"
	//errors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	//types "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	appsv1alpha1 "github.com/rforberger/demo-operator/api/v1alpha1"
	apimachineryjson "k8s.io/apimachinery/pkg/util/json"
	"sigs.k8s.io/kustomize/api/krusty"
	kustomize_types "sigs.k8s.io/kustomize/api/types"
	"sigs.k8s.io/kustomize/kyaml/filesys"

	//apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	//yaml "go.yaml.in/yaml/v4"

	"sigs.k8s.io/kustomize/api/resource"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	//utils "github.com/rforberger/demo-operator/test/utils"
	// +kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var (
	ctx       context.Context
	cancel    context.CancelFunc
	testEnv   *envtest.Environment
	cfg       *rest.Config
	k8sClient client.Client
)

func debug(str string) {
	_, _ = fmt.Fprintf(GinkgoWriter, "warning: %v\n", str)
}

func TestControllers(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Controller Suite")
}

var _ = BeforeSuite(func() {

	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	ctx, cancel = context.WithCancel(context.TODO())

	var err error
	err = appsv1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	// +kubebuilder:scaffold:scheme
	By("bootstrapping test environment")
	paths := []string{"..", "..", "kubernetes-manifests", "github.com", "sigs.k8s.io", "gateway-api", "config", "crd"}
	testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{
			filepath.Join("..", "..", "config", "crd", "bases"),
			//filepath.Join("..", "..", "kubernetes-manifests", "github.com", "sigs.k8s.io", "gateway-api"),
		},
		ErrorIfCRDPathMissing: false,

		// Redirects api-server & etcd stdout/stderr directly to your terminal
		AttachControlPlaneOutput: false,

		// Load CRDs from an external path
		// CRDs: LoadCRDS(paths),
	}

	// Retrieve the first found binary directory to allow running tests from IDEs
	//if getFirstFoundEnvTestBinaryDir() != "" {
	//	testEnv.BinaryAssetsDirectory = getFirstFoundEnvTestBinaryDir()
	//}

	// cfg is defined in this file globally.
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	// =========================================================================
	// 4. THE CALL: Pass the setup context and the newly created client instance
	// =========================================================================
	bootstrapClusterResources(ctx, k8sClient, paths)
})

/*
func LoadCRDS(paths []string) []*apiextensionsv1.CustomResourceDefinition {
	kOpts := krusty.MakeDefaultOptions()
	kOpts.PluginConfig = kustomize_types.EnabledPluginConfig(kustomize_types.BploUndefined)
	kOpts.PluginConfig.HelmConfig.Command = "helm"
	k := krusty.MakeKustomizer(kOpts)
	m, err := k.Run(filesys.FileSystemOrOnDisk{}, filepath.Join(paths...))
	Expect(err).To(Succeed())

	resources := m.Resources()

	crds := make([]*apiextensionsv1.CustomResourceDefinition, len(resources))
	for i := range resources {
		bytes, err := resources[i].MarshalJSON()
		Expect(err).To(Succeed())

		crd := &apiextensionsv1.CustomResourceDefinition{}
		err = apimachineryjson.Unmarshal(bytes, crd)
		Expect(err).To(Succeed())

		crds[i] = crd
	}

	return crds
}
*/

func ResourceToUnstructured(res *resource.Resource) (*unstructured.Unstructured, error) {
	// 1. Get the resource as a map
	resMap, err := res.Map()
	if err != nil {
		return nil, err
	}

	// 2. Wrap it in Unstructured — which implements client.Object
	u := &unstructured.Unstructured{Object: resMap}
	return u, nil
}

func applyYamlManifestFromFilepaths(ctx context.Context, c client.Client, paths []string) []*resource.Resource {
	kOpts := krusty.MakeDefaultOptions()
	kOpts.PluginConfig = kustomize_types.EnabledPluginConfig(kustomize_types.BploUndefined)
	kOpts.PluginConfig.HelmConfig.Command = "helm"
	k := krusty.MakeKustomizer(kOpts)
	m, err := k.Run(filesys.FileSystemOrOnDisk{}, filepath.Join(paths...)) // type ResMap, type error
	Expect(err).NotTo(HaveOccurred())

	resources := m.Resources() // type *resource.Resource

	objects := make([]*resource.Resource, len(resources))
	for i := range resources {
		bytes, err := resources[i].MarshalJSON()
		Expect(err).To(Succeed())

		object := &resource.Resource{}
		err = apimachineryjson.Unmarshal(bytes, object)
		Expect(err).To(Succeed())

		objects[i] = object
	}

	return objects
}

func bootstrapClusterResources(ctx context.Context, c client.Client, paths []string) {
	By("Bootstrapping native and custom test structures")

	// --- TYPE A: Native Go Structs (For standard resources like Namespaces/ConfigMaps) ---
	/*
		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{Name: "my-test-namespace"},
		}
		Expect(c.Create(ctx, ns)).To(Succeed())
	*/

	/*
		globalConfig := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "operator-settings",
				//Namespace: "my-test-namespace",
			},
			Data: map[string]string{"max-workers": "10"},
		}
		Expect(c.Create(ctx, globalConfig)).To(Succeed())
	*/

	// --- TYPE B: Multi-Document Raw YAML manifests (For third-party integrations) ---
	// manifestPath := filepath.Join(paths)
	objects := applyYamlManifestFromFilepaths(ctx, c, paths)

	for _, object := range objects {

		//fmt.Println(object)
		u, err := ResourceToUnstructured(object)
		Expect(err).NotTo(HaveOccurred())

		fmt.Println(u)
		// Safely construct the object in envtest's etcd database
		Expect(c.Create(ctx, u)).To(Succeed(), "Failed to auto-apply resource %s/%s", u.GetNamespace(), u.GetName())
	}

}

////////////
// TESTS //
///////////

// Gateway controller test
var _ = Describe("Gateway controller", func() {
	Context("Gateway controller test", func() {
		It("should build the correct gateway", func() {

			// Test for a Gateway
			//

			ctx := context.Background()

			desired_port := 8080
			port := int32(desired_port)

			// CR
			app := &appsv1alpha1.DemoApp{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-resource",
					Namespace: "default",
				},
				Spec: appsv1alpha1.DemoAppSpec{
					Deployments: []appsv1alpha1.DeploymentSpec{},
					Gateway: &appsv1alpha1.GatewayRef{
						Name: "my-gateway",
						Port: &port,
						HTTP: appsv1alpha1.HTTPRouteSpec{
							Hostnames: []string{},
						},
					},
				},
			}

			// CR Test
			reconciler_cr := &DemoAppReconciler{}
			gw := reconciler_cr.desiredGateway(app)

			Expect(gw).NotTo(BeNil())
			Expect(gw.Name).To(Equal(app.Name + "-gateway"))
			Expect(gw.Spec.Listeners).To(HaveLen(1))
			Expect(gw.Spec.Listeners[0].Port).To(Equal(gatewayv1.PortNumber(desired_port)))

			// K8sclient
			Expect(k8sClient.Create(ctx, app)).To(Succeed())

			// Clean up after the test
			DeferCleanup(func() {
				Expect(k8sClient.Delete(ctx, app)).To(Succeed())
			})

			// Wait for the reconciler to do something observable
			Eventually(func(g Gomega) {
				updated := &appsv1alpha1.DemoApp{}
				g.Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(app), updated)).To(Succeed())

				// Expand conditions here
				////g.Expect(updated.Status.Accepted).To(Equal(true))
			}).Should(Succeed())

			//
			// 2. Instantiate your reconciler the same way main.go or suite_test.go does
			reconciler := &DemoAppReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			// 3. Call the function directly
			gateway := reconciler.desiredGateway(app)

			// Build gateway ObjectKey
			key := client.ObjectKey{
				Name:      "my-gateway-gateway",
				Namespace: "default",
			}
			// Wait for the reconciler to do something observable
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, key, gateway)).To(Succeed())

				// Expand conditions here
				//	g.Expect(gateway.Status.Accepted).To(Equal(true))
			}).Should(Succeed())
		})
	})
})

// ///////////////
// AfterSuite //
// //////////////
var _ = AfterSuite(func() {
	By("tearing down the test environment")
	cancel()
	Eventually(func() error {
		return testEnv.Stop()
	}, time.Minute, time.Second).Should(Succeed())
})

// getFirstFoundEnvTestBinaryDir locates the first binary in the specified path.
// ENVTEST-based tests depend on specific binaries, usually located in paths set by
// controller-runtime. When running tests directly (e.g., via an IDE) without using
// Makefile targets, the 'BinaryAssetsDirectory' must be explicitly configured.
//
// This function streamlines the process by finding the required binaries, similar to
// setting the 'KUBEBUILDER_ASSETS' environment variable. To ensure the binaries are
// properly set up, run 'make setup-envtest' beforehand.
/*
func getFirstFoundEnvTestBinaryDir() string {
	basePath := filepath.Join("..", "..", "bin", "k8s")
	entries, err := os.ReadDir(basePath)
	if err != nil {
		logf.Log.Error(err, "Failed to read directory", "path", basePath)
		return ""
	}
	for _, entry := range entries {
		if entry.IsDir() {
			return filepath.Join(basePath, entry.Name())
		}
	}
	return ""
}
*/
