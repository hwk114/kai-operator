/*
Copyright 2026 KAI Team.

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

package devtask

import (
	"context"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	gatewayapiv1 "sigs.k8s.io/gateway-api/apis/v1"

	kaiiov1alpha1 "github.com/hwk114/kai/api/v1alpha1"
)

const (
	RouteFinalizer = "kai.io/route"
)

func BuildHTTPRoute(task *kaiiov1alpha1.DevTask, serviceName string, servicePort int32) *gatewayapiv1.HTTPRoute {
	framework := GetFramework(task)
	path := GetRoutePath(task)
	gatewayName := GetGatewayName(task)
	gatewayNamespace := GetGatewayNamespace(task)
	hostnames := GetHostnames(task)

	pathValue := path
	pathType := gatewayapiv1.PathMatchPathPrefix

	route := &gatewayapiv1.HTTPRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      GetRouteName(task),
			Namespace: task.Namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":       "kai",
				"app.kubernetes.io/managed-by": "kai-operator",
				"kai.io/devtask":               task.Name,
				"framework":                    framework,
			},
		},
		Spec: gatewayapiv1.HTTPRouteSpec{
			Hostnames: hostnames,
			CommonRouteSpec: gatewayapiv1.CommonRouteSpec{
				ParentRefs: []gatewayapiv1.ParentReference{
					{
						Group:     ptrTo(gatewayapiv1.Group("gateway.networking.k8s.io")),
						Kind:      ptrTo(gatewayapiv1.Kind("Gateway")),
						Name:      gatewayapiv1.ObjectName(gatewayName),
						Namespace: ptrTo(gatewayapiv1.Namespace(gatewayNamespace)),
					},
				},
			},
			Rules: []gatewayapiv1.HTTPRouteRule{
				{
					Matches: []gatewayapiv1.HTTPRouteMatch{
						{
							Path: &gatewayapiv1.HTTPPathMatch{
								Type:  ptrTo(pathType),
								Value: &pathValue,
							},
						},
					},
					BackendRefs: []gatewayapiv1.HTTPBackendRef{
						{
							BackendRef: gatewayapiv1.BackendRef{
								BackendObjectReference: gatewayapiv1.BackendObjectReference{
									Group: ptrTo(gatewayapiv1.Group("")),
									Kind:  ptrTo(gatewayapiv1.Kind("Service")),
									Name:  gatewayapiv1.ObjectName(serviceName),
									Port:  ptrTo(gatewayapiv1.PortNumber(servicePort)),
								},
							},
						},
					},
				},
			},
		},
	}

	// Add URL rewrite filter only for code-server and novnc
	if framework == "codeserver" || framework == "novnc" {
		route.Spec.Rules[0].Filters = []gatewayapiv1.HTTPRouteFilter{
			{
				Type: gatewayapiv1.HTTPRouteFilterURLRewrite,
				URLRewrite: &gatewayapiv1.HTTPURLRewriteFilter{
					Path: &gatewayapiv1.HTTPPathModifier{
						Type:               gatewayapiv1.PrefixMatchHTTPPathModifier,
						ReplacePrefixMatch: ptrTo("/"),
					},
				},
			},
		}
	}

	// Add WebSocket route for novnc
	if framework == "novnc" {
		wsPath := pathValue + "websockify"
		wsRule := gatewayapiv1.HTTPRouteRule{
			Matches: []gatewayapiv1.HTTPRouteMatch{
				{
					Path: &gatewayapiv1.HTTPPathMatch{
						Type:  ptrTo(gatewayapiv1.PathMatchPathPrefix),
						Value: &wsPath,
					},
				},
			},
			BackendRefs: []gatewayapiv1.HTTPBackendRef{
				{
					BackendRef: gatewayapiv1.BackendRef{
						BackendObjectReference: gatewayapiv1.BackendObjectReference{
							Group: ptrTo(gatewayapiv1.Group("")),
							Kind:  ptrTo(gatewayapiv1.Kind("Service")),
							Name:  gatewayapiv1.ObjectName(serviceName),
							Port:  ptrTo(gatewayapiv1.PortNumber(servicePort)),
						},
					},
				},
			},
			Filters: []gatewayapiv1.HTTPRouteFilter{
				{
					Type: gatewayapiv1.HTTPRouteFilterURLRewrite,
					URLRewrite: &gatewayapiv1.HTTPURLRewriteFilter{
						Path: &gatewayapiv1.HTTPPathModifier{
							Type:               gatewayapiv1.PrefixMatchHTTPPathModifier,
							ReplacePrefixMatch: ptrTo("/"),
						},
					},
				},
			},
		}
		route.Spec.Rules = append(route.Spec.Rules, wsRule)
	}

	return route
}

func CreateHTTPRoute(ctx context.Context, c client.Client, scheme *runtime.Scheme, task *kaiiov1alpha1.DevTask, serviceName string, servicePort int32) error {
	if !NeedsExternalRouting(task) {
		return nil
	}

	route := BuildHTTPRoute(task, serviceName, servicePort)

	if err := controllerutil.SetControllerReference(task, route, scheme); err != nil {
		return err
	}

	controllerutil.AddFinalizer(route, RouteFinalizer)

	existingRoute := &gatewayapiv1.HTTPRoute{}
	err := c.Get(ctx, types.NamespacedName{Name: route.Name, Namespace: route.Namespace}, existingRoute)
	if err != nil && errors.IsNotFound(err) {
		if err := c.Create(ctx, route); err != nil {
			return err
		}
	}

	return nil
}

func CleanupHTTPRoute(ctx context.Context, c client.Client, task *kaiiov1alpha1.DevTask) error {
	routeName := GetRouteName(task)

	route := &gatewayapiv1.HTTPRoute{}
	if err := c.Get(ctx, types.NamespacedName{Name: routeName, Namespace: task.Namespace}, route); err == nil {
		if controllerutil.ContainsFinalizer(route, RouteFinalizer) {
			controllerutil.RemoveFinalizer(route, RouteFinalizer)
			if err := c.Update(ctx, route); err != nil && !errors.IsNotFound(err) {
				return err
			}
		}
		zero := int64(0)
		if err := c.Delete(ctx, route, &client.DeleteOptions{GracePeriodSeconds: &zero}); err != nil && !errors.IsNotFound(err) {
			return err
		}
	}

	routeList := &gatewayapiv1.HTTPRouteList{}
	if err := c.List(ctx, routeList, client.MatchingLabels{"kai.io/devtask": task.Name}, client.InNamespace(task.Namespace)); err == nil {
		for _, route := range routeList.Items {
			if controllerutil.ContainsFinalizer(&route, RouteFinalizer) {
				controllerutil.RemoveFinalizer(&route, RouteFinalizer)
				if err := c.Update(ctx, &route); err != nil && !errors.IsNotFound(err) {
					return err
				}
			}
			zero := int64(0)
			if err := c.Delete(ctx, &route, &client.DeleteOptions{GracePeriodSeconds: &zero}); err != nil && !errors.IsNotFound(err) {
				return err
			}
		}
	}

	return nil
}

func ptrTo[T any](v T) *T {
	return &v
}

func EnsureHTTPRoute(ctx context.Context, c client.Client, scheme *runtime.Scheme, task *kaiiov1alpha1.DevTask) error {
	if !NeedsExternalRouting(task) {
		return nil
	}

	routeName := GetRouteName(task)

	existingRoute := &gatewayapiv1.HTTPRoute{}
	if err := c.Get(ctx, types.NamespacedName{Name: routeName, Namespace: task.Namespace}, existingRoute); err == nil {
		return nil
	}

	serviceName := GetServiceNameForTask(task)
	servicePort := GetServicePortForTask(task)

	route := BuildHTTPRoute(task, serviceName, servicePort)

	if err := controllerutil.SetControllerReference(task, route, scheme); err != nil {
		return err
	}

	controllerutil.AddFinalizer(route, RouteFinalizer)

	if err := c.Create(ctx, route); err != nil && !errors.IsAlreadyExists(err) {
		return err
	}

	return nil
}
