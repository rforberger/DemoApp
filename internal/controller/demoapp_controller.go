/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	v1alpha1 "github.com/rforberger/demo-operator/api/v1alpha1"
)

// DemoAppReconciler reconciles a DemoApp object
type DemoAppReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=apps.example.com,resources=demoapps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps.example.com,resources=demoapps/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps.example.com,resources=demoapps/finalizers,verbs=update

// ⬇⬇⬇ DAS FEHLTE ⬇⬇⬇
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the DemoApp object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.22.4/pkg/reconcile
func (r *DemoAppReconciler) Reconcile(
	ctx context.Context,
	req ctrl.Request,
) (ctrl.Result, error) {

	logger := logf.FromContext(ctx)

	// 1. Custom Resource laden
	var demo v1alpha1.DemoApp
	if err := r.Get(ctx, req.NamespacedName, &demo); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// 2. Deployment-Name festlegen
	deployName := demo.Name + "-deployment"

	// 3. Deployment suchen
	var deploy appsv1.Deployment
	err := r.Get(ctx, types.NamespacedName{
		Name:      deployName,
		Namespace: demo.Namespace,
	}, &deploy)

	// 3.1 ReadinessProbe setzen
    var readinessProbe *corev1.Probe

    if demo.Spec.ReadinessProbe != nil && demo.Spec.ReadinessProbe.HTTPGet != nil {
        rp := demo.Spec.ReadinessProbe

        readinessProbe = &corev1.Probe{
            ProbeHandler: corev1.ProbeHandler{
                HTTPGet: &corev1.HTTPGetAction{
                    Path: rp.HTTPGet.Path,
                    Port: intstr.FromInt(int(rp.HTTPGet.Port)),
                    Scheme: func() corev1.URIScheme {
                        if rp.HTTPGet.Scheme != nil {
                            return *rp.HTTPGet.Scheme
                        }
                        return corev1.URISchemeHTTP
                    }(),
                },
            },
        }

        if rp.InitialDelaySeconds != nil {
            readinessProbe.InitialDelaySeconds = *rp.InitialDelaySeconds
        }
        if rp.PeriodSeconds != nil {
            readinessProbe.PeriodSeconds = *rp.PeriodSeconds
        }
    }

	// 4. Deployment existiert nicht → erstellen
	if apierrors.IsNotFound(err) {
		deploy = appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      deployName,
				Namespace: demo.Namespace,
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: demo.Spec.Replicas,
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"app": demo.Name,
					},
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"app": demo.Name,
						},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  "app",
								Image: demo.Spec.Image,
								Ports: []corev1.ContainerPort{
									{ContainerPort: 80},
								},
                                ReadinessProbe: readinessProbe,
							},
						},
					},
				},
			},
		}

		// OwnerReference setzen (SEHR wichtig!)
		if err := ctrl.SetControllerReference(&demo, &deploy, r.Scheme); err != nil {
			return ctrl.Result{}, err
		}

		logger.Info("Creating Deployment", "name", deployName)
		return ctrl.Result{}, r.Create(ctx, &deploy)
	}

	if err != nil {
		return ctrl.Result{}, err
	}

	// 5. Status aktualisieren
	demo.Status.AvailableReplicas = deploy.Status.AvailableReplicas
	if err := r.Status().Update(ctx, &demo); err != nil {
		logger.Error(err, "unable to update status")
	}

	return ctrl.Result{}, nil
}

func (r *DemoAppReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.DemoApp{}).
		Owns(&appsv1.Deployment{}).
		Complete(r)
}

