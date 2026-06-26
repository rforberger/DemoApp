package controller

import (
	"context"
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
	testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{
			filepath.Join("..", "..", "config", "crd", "bases"),
			filepath.Join("..", "..", "vendor", "sigs.k8s.io", "gateway-api", "config", "crd", "standard"),
		},
		ErrorIfCRDPathMissing: false,
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
})

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
