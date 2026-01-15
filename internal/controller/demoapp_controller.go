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
	"k8s.io/apimachinery/pkg/util/intstr"
	v1alpha1 "github.com/rforberger/demo-operator/api/v1alpha1"
)

// DemoAppReconciler reconciles a DemoApp object
type DemoAppReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func buildReadinessProbe(p *ReadinessProbeSpec) *corev1.Probe {
	if p == nil {
		return nil
	}

	probe := &corev1.Probe{
		InitialDelaySeconds: p.InitialDelaySeconds,
		PeriodSeconds:       p.PeriodSeconds,
	}

	// HTTP GET Probe
	if p.Path != "" && p.Port > 0 {
		probe.ProbeHandler = corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Path: p.Path,
				Port: intstr.FromInt(int(p.Port)),
			},
		}
	}

	return probe
}

func buildContainers(specs []ContainerSpec) []corev1.Container {
	containers := make([]corev1.Container, 0, len(specs))

	for _, c := range specs {
		container := corev1.Container{
			Name:  c.Name,
			Image: c.Image,
		}

		// ReadinessProbe (optional)
		if c.ReadinessProbe != nil {
			container.ReadinessProbe = buildReadinessProbe(c.ReadinessProbe)
		}

		containers = append(containers, container)
	}

	return containers
}

func buildDeploymentStrategy(s *DeploymentStrategySpec) appsv1.DeploymentStrategy {
	if s == nil {
		return appsv1.DeploymentStrategy{
			Type: appsv1.RollingUpdateDeploymentStrategyType,
		}
	}

	strategy := appsv1.DeploymentStrategy{
		Type: appsv1.DeploymentStrategyType(s.Type),
	}

	if s.Type == "RollingUpdate" && s.RollingUpdate != nil {
		strategy.RollingUpdate = &appsv1.RollingUpdateDeployment{
			MaxSurge:       intstr.FromInt(int(*s.RollingUpdate.MaxSurge)),
			MaxUnavailable: intstr.FromInt(int(*s.RollingUpdate.MaxUnavailable)),
		}
	}

	return strategy
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

    for _, dep := range demo.Spec.Deployments {

        deploymentName := fmt.Sprintf("%s-%s", demo.Name, dep.Name)

        deployment := appsv1.Deployment{
            ObjectMeta: metav1.ObjectMeta{
                Name:      deploymentName,
                Namespace: demo.Namespace,
            },
            Spec: appsv1.DeploymentSpec{
                Replicas: dep.Replicas,
                Selector: &metav1.LabelSelector{
                    MatchLabels: map[string]string{
                        "app": deploymentName,
                    },
                },
                Template: corev1.PodTemplateSpec{
                    ObjectMeta: metav1.ObjectMeta{
                        Labels: map[string]string{
                            "app": deploymentName,
                        },
                    },
                    Spec: corev1.PodSpec{
                        Containers: buildContainers(dep.Containers),
                    },
                },
            },
        }

        // OwnerReference setzen!
        ctrl.SetControllerReference(&demo, &deployment, r.Scheme)

        // Create / Update
    }

	return ctrl.Result{}, nil
}

func (r *DemoAppReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.DemoApp{}).
		Owns(&appsv1.Deployment{}).
		Complete(r)
}

