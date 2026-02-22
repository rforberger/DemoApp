package controller

// +kubebuilder:rbac:groups=apps.example.com,resources=demoapps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps.example.com,resources=demoapps/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=gateways,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=gateways/status,verbs=get;update;patch

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	demov1alpha1 "github.com/rforberger/demo-operator/api/v1alpha1"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
    //gatewayv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
)


// DemoAppReconciler reconciles a DemoApp object
type DemoAppReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func toHostnames(h []string) []gatewayv1.Hostname {
    out := make([]gatewayv1.Hostname, 0, len(h))
    for _, v := range h {
        out = append(out, gatewayv1.Hostname(v))
    }
    return out
}

func (r *DemoAppReconciler) desiredHTTPRoute(app *demov1alpha1.DemoApp) *gatewayv1.HTTPRoute {
    gwNS := app.Spec.Gateway.Namespace
    if gwNS == "" {
        gwNS = app.Namespace
    }

    //pathType := gatewayv1beta1.PathMatchPathPrefix
    pathType := gatewayv1.PathMatchType("PathPrefix")
    pathValue := "/"
    if app.Spec.HTTP.Path != "" {
        pathValue = app.Spec.HTTP.Path
    }

    return &gatewayv1.HTTPRoute{
        ObjectMeta: metav1.ObjectMeta{
            Name:      app.Name,
            Namespace: app.Namespace,
            Labels: map[string]string{
                "app": app.Name,
            },
        },
        Spec: gatewayv1.HTTPRouteSpec{
            ParentRefs: []gatewayv1.ParentReference{
                {
                    Name:      gatewayv1.ObjectName(app.Spec.Gateway.Name),
                    Namespace: (*gatewayv1.Namespace)(&gwNS),
                },
            },
            Hostnames: toHostnames(app.Spec.HTTP.Hostnames),
            Rules: []gatewayv1.HTTPRouteRule{
                {
                    Matches: []gatewayv1.HTTPRouteMatch{
                        {
                            Path: &gatewayv1.HTTPPathMatch{
                                Type:  &pathType,
                                Value: &pathValue,
                            },
                        },
                    },
                    BackendRefs: []gatewayv1.HTTPBackendRef{
                        {
                            BackendRef: gatewayv1.BackendRef{
                                BackendObjectReference: gatewayv1.BackendObjectReference{
                                    Name: gatewayv1.ObjectName(app.Name),
                                    Port: ptr.To,
                                },
                            },
                        },
                    },
                },
            },
        },
    }
}

func (r *DemoAppReconciler) desiredGateway(app *demov1alpha1.DemoApp) *gatewayv1.Gateway {
    return &gatewayv1.Gateway{
        ObjectMeta: metav1.ObjectMeta{
            Name:      app.Name + "-gateway",
            Namespace: app.Namespace,
        },
        Spec: gatewayv1.GatewaySpec{
            GatewayClassName: gatewayv1.ObjectName("nginx"),
            Listeners: []gatewayv1.Listener{
                {
                    Name:     app.Gateway.Name,
                    Port:     app.Gateway.Port,
                    Protocol: gatewayv1.HTTPProtocolType,
                },
            },
        },
    }
}


func (r *DemoAppReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var demoApp demov1alpha1.DemoApp
	if err := r.Get(ctx, req.NamespacedName, &demoApp); err != nil {
		logger.Error(err, "unable to fetch DemoApp")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	for _, d := range demoApp.Spec.Deployments {
		deploy := buildDeployment(d, demoApp.Namespace)
		if err := r.Client.Create(ctx, deploy); err != nil {
			logger.Error(err, "failed to create deployment", "deployment", d.Name)
			continue
		}
		logger.Info("Deployment created", "deployment", d.Name)
	}

    // Gateway API
    var gw gatewayv1.Gateway
    err := r.Get(ctx,
        client.ObjectKey{
            Name:      demoApp.Name + "-gateway",
            Namespace: demoApp.Namespace,
        },
        &gw,
    )

    if apierrors.IsNotFound(err) {
        gw = *r.desiredGateway(&demoApp)

        if err := ctrl.SetControllerReference(&demoApp, &gw, r.Scheme); err != nil {
            return ctrl.Result{}, err
        }

        if err := r.Create(ctx, &gw); err != nil {
            return ctrl.Result{}, err
        }
    }

    // Desired HTTP HTTPRoute
    var route gatewayv1.HTTPRoute
    err := r.Get(ctx, types.NamespacedName{
        Name:      demoApp.Name,
        Namespace: demoApp.Namespace,
    }, &route)

    desired := r.desiredHTTPRoute(demoApp)
    controllerutil.SetControllerReference(demoApp, desired, r.Scheme)

    if apierrors.IsNotFound(err) {
        return ctrl.Result{}, r.Create(ctx, desired)
    }

    if err != nil {
        return ctrl.Result{}, err
    }

    patch := client.MergeFrom(route.DeepCopy())
    route.Spec = desired.Spec
    route.Labels = desired.Labels

	return ctrl.Result{}, r.Patch(ctx, &route, patch)
}

func buildDeployment(d demov1alpha1.DeploymentSpec, namespace string) *appsv1.Deployment {
	replicas := int32(1)
	if d.Replicas != nil {
		replicas = *d.Replicas
	}

	labels := map[string]string{
		"app":  d.Name,
		"name": d.Name,
		"tier": "backend",
	}

	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      d.Name,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels, // ⚡ Fix here
				},
				Spec: corev1.PodSpec{
					Containers: buildContainers(d.Containers),
				},
			},
		},
	}

	// Deployment Strategy
	if d.Strategy != nil && d.Strategy.Type != "" {
		strat := appsv1.DeploymentStrategy{
			Type: appsv1.DeploymentStrategyType(d.Strategy.Type),
		}
		if d.Strategy.RollingUpdate != nil {
			ru := &appsv1.RollingUpdateDeployment{}
			if d.Strategy.RollingUpdate.MaxSurge != nil {
				ms := intstr.FromInt(int(*d.Strategy.RollingUpdate.MaxSurge))
				ru.MaxSurge = &ms
			}
			if d.Strategy.RollingUpdate.MaxUnavailable != nil {
				mu := intstr.FromInt(int(*d.Strategy.RollingUpdate.MaxUnavailable))
				ru.MaxUnavailable = &mu
			}
			strat.RollingUpdate = ru
		}
		deploy.Spec.Strategy = strat
	}

	return deploy
}

func buildContainers(specs []demov1alpha1.ContainerSpec) []corev1.Container {
	containers := make([]corev1.Container, 0, len(specs))
	for _, c := range specs {
		container := corev1.Container{
			Name:  c.Name,
			Image: c.Image,
		}
		if c.Resources != nil {
			container.Resources = buildResources(c.Resources)
		}
		if c.ReadinessProbe != nil {
			container.ReadinessProbe = buildReadinessProbe(c.ReadinessProbe)
		}
		containers = append(containers, container)
	}
	return containers
}

func buildResources(r *demov1alpha1.ResourceSpec) corev1.ResourceRequirements {
	res := corev1.ResourceRequirements{
		Requests: corev1.ResourceList{},
		Limits:   corev1.ResourceList{},
	}
	if r.Requests != nil {
		if r.Requests.CPU != "" {
			res.Requests[corev1.ResourceCPU] = resource.MustParse(r.Requests.CPU)
		}
		if r.Requests.Memory != "" {
			res.Requests[corev1.ResourceMemory] = resource.MustParse(r.Requests.Memory)
		}
	}
	if r.Limits != nil {
		if r.Limits.CPU != "" {
			res.Limits[corev1.ResourceCPU] = resource.MustParse(r.Limits.CPU)
		}
		if r.Limits.Memory != "" {
			res.Limits[corev1.ResourceMemory] = resource.MustParse(r.Limits.Memory)
		}
	}
	return res
}

func buildReadinessProbe(p *demov1alpha1.ReadinessProbeSpec) *corev1.Probe {
	if p == nil || p.HTTPGet == nil {
		return nil
	}
	scheme := corev1.URISchemeHTTP
	if p.HTTPGet.Scheme != nil {
		scheme = *p.HTTPGet.Scheme
	}
	initialDelay := int32(5)
	if p.InitialDelaySeconds != nil {
		initialDelay = *p.InitialDelaySeconds
	}
	period := int32(10)
	if p.PeriodSeconds != nil {
		period = *p.PeriodSeconds
	}

	return &corev1.Probe{
		InitialDelaySeconds: initialDelay,
		PeriodSeconds:       period,
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Path:   p.HTTPGet.Path,
				Port:   intstr.FromInt(int(p.HTTPGet.Port)),
				Scheme: scheme,
			},
		},
	}
}

// SetupWithManager registers the controller with the manager
func (r *DemoAppReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&demov1alpha1.DemoApp{}).
		Owns(&appsv1.Deployment{}).
		Complete(r)
}
