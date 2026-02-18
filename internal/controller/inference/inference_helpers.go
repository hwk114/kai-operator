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

package inference

import (
	"context"
	"fmt"
	"net"

	kaiiov1alpha1 "github.com/hwk114/kai/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	gatewayapiv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func getServiceName(inference *kaiiov1alpha1.InferenceTask) string {
	framework := inference.GetFramework()
	if framework == "" {
		framework = GetEngine(inference)
	}
	return fmt.Sprintf("inf-%s-%s", framework, inference.Name)
}

func ensureService(ctx context.Context, c client.Client, scheme *runtime.Scheme, inference *kaiiov1alpha1.InferenceTask, svcName string, namespace string, port int32, labels map[string]string) error {
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      svcName,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Selector: labels,
			Ports: []corev1.ServicePort{
				{
					Name:     "http",
					Port:     port,
					Protocol: corev1.ProtocolTCP,
				},
			},
			Type: corev1.ServiceTypeClusterIP,
		},
	}

	if err := controllerutil.SetControllerReference(inference, svc, scheme); err != nil {
		return err
	}

	existingSvc := &corev1.Service{}
	err := c.Get(ctx, types.NamespacedName{Name: svcName, Namespace: namespace}, existingSvc)
	if err != nil && errors.IsNotFound(err) {
		if err := c.Create(ctx, svc); err != nil {
			return err
		}
	}

	return nil
}

func deleteService(ctx context.Context, c client.Client, svcName string, namespace string) error {
	svc := &corev1.Service{}
	if err := c.Get(ctx, types.NamespacedName{Name: svcName, Namespace: namespace}, svc); err == nil {
		if err := c.Delete(ctx, svc); err != nil && !errors.IsNotFound(err) {
			return err
		}
	}
	return nil
}

func ensureHTTPRoute(ctx context.Context, c client.Client, scheme *runtime.Scheme, inference *kaiiov1alpha1.InferenceTask, serviceName string, servicePort int32) error {
	if !inference.NeedsExternalRouting() {
		return nil
	}

	route := buildHTTPRoute(inference, serviceName, servicePort)

	if err := controllerutil.SetControllerReference(inference, route, scheme); err != nil {
		return err
	}

	existingRoute := &gatewayapiv1.HTTPRoute{}
	err := c.Get(ctx, types.NamespacedName{Name: route.Name, Namespace: route.Namespace}, existingRoute)
	if err != nil && errors.IsNotFound(err) {
		if err := c.Create(ctx, route); err != nil {
			return err
		}
	}

	return nil
}

func cleanupHTTPRoute(ctx context.Context, c client.Client, inference *kaiiov1alpha1.InferenceTask) error {
	framework := inference.GetFramework()
	if framework == "" {
		framework = GetEngine(inference)
	}
	routeName := fmt.Sprintf("inf-%s-%s", framework, inference.Name)

	route := &gatewayapiv1.HTTPRoute{}
	if err := c.Get(ctx, types.NamespacedName{Name: routeName, Namespace: inference.Namespace}, route); err == nil {
		if controllerutil.ContainsFinalizer(route, "kai.io/inference-route") {
			controllerutil.RemoveFinalizer(route, "kai.io/inference-route")
			if err := c.Update(ctx, route); err != nil && !errors.IsNotFound(err) {
				return err
			}
		}
		zero := int64(0)
		if err := c.Delete(ctx, route, &client.DeleteOptions{GracePeriodSeconds: &zero}); err != nil && !errors.IsNotFound(err) {
			return err
		}
	}

	return nil
}

func buildHTTPRoute(inference *kaiiov1alpha1.InferenceTask, serviceName string, servicePort int32) *gatewayapiv1.HTTPRoute {
	framework := inference.GetFramework()
	if framework == "" {
		framework = GetEngine(inference)
	}
	path := inference.GetRoutePath()
	gatewayName := inference.GetGatewayName()
	gatewayNamespace := inference.GetGatewayNamespace()

	pathValue := path
	pathType := gatewayapiv1.PathMatchPathPrefix

	route := &gatewayapiv1.HTTPRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("inf-%s-%s", framework, inference.Name),
			Namespace: inference.Namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":       "kai",
				"app.kubernetes.io/managed-by": "kai-operator",
				"kai.io/inference":             inference.Name,
				"framework":                    framework,
			},
		},
		Spec: gatewayapiv1.HTTPRouteSpec{
			CommonRouteSpec: gatewayapiv1.CommonRouteSpec{
				ParentRefs: []gatewayapiv1.ParentReference{
					{
						Group:     ptrToGateway(gatewayapiv1.Group("gateway.networking.k8s.io")),
						Kind:      ptrToGateway(gatewayapiv1.Kind("Gateway")),
						Name:      gatewayapiv1.ObjectName(gatewayName),
						Namespace: ptrToGateway(gatewayapiv1.Namespace(gatewayNamespace)),
					},
				},
			},
			Rules: []gatewayapiv1.HTTPRouteRule{
				{
					Matches: []gatewayapiv1.HTTPRouteMatch{
						{
							Path: &gatewayapiv1.HTTPPathMatch{
								Type:  ptrToGateway(pathType),
								Value: &pathValue,
							},
						},
					},
					Filters: []gatewayapiv1.HTTPRouteFilter{
						{
							Type: gatewayapiv1.HTTPRouteFilterURLRewrite,
							URLRewrite: &gatewayapiv1.HTTPURLRewriteFilter{
								Path: &gatewayapiv1.HTTPPathModifier{
									Type:               gatewayapiv1.PrefixMatchHTTPPathModifier,
									ReplacePrefixMatch: ptrToGateway("/"),
								},
							},
						},
					},
					BackendRefs: []gatewayapiv1.HTTPBackendRef{
						{
							BackendRef: gatewayapiv1.BackendRef{
								BackendObjectReference: gatewayapiv1.BackendObjectReference{
									Group: ptrToGateway(gatewayapiv1.Group("")),
									Kind:  ptrToGateway(gatewayapiv1.Kind("Service")),
									Name:  gatewayapiv1.ObjectName(serviceName),
									Port:  ptrToGateway(gatewayapiv1.PortNumber(servicePort)),
								},
							},
						},
					},
				},
			},
		},
	}

	return route
}

func ptrToGateway[T any](v T) *T {
	return &v
}

func getExternalURL(ctx context.Context, c client.Client, inference *kaiiov1alpha1.InferenceTask) string {
	if !inference.NeedsExternalRouting() {
		return ""
	}

	cfg := inference.GetRoutingConfig()
	gatewayName := inference.GetGatewayName()
	gatewayNamespace := inference.GetGatewayNamespace()
	path := inference.GetRoutePath()

	gatewayIP, gatewayPort, err := getGatewayAddressAndPort(ctx, c, gatewayName, gatewayNamespace, cfg)
	if err != nil {
		return fmt.Sprintf("http://%s.%s.svc%s", gatewayName, gatewayNamespace, path)
	}

	logger := log.FromContext(ctx)
	logger.Info("getExternalURL", "gatewayIP", gatewayIP, "gatewayPort", gatewayPort, "path", path)

	scheme := "http"
	if gatewayPort == 443 {
		scheme = "https"
	}

	if gatewayPort == 80 || gatewayPort == 443 {
		return fmt.Sprintf("%s://%s%s", scheme, gatewayIP, path)
	}
	return fmt.Sprintf("%s://%s:%d%s", scheme, gatewayIP, gatewayPort, path)
}

func getGatewayAddressAndPort(ctx context.Context, c client.Client, gatewayName, gatewayNamespace string, cfg *kaiiov1alpha1.RoutingSpec) (string, int32, error) {
	hostIP, err := getGatewayHostIPInference(ctx, c, gatewayName, gatewayNamespace)
	if err != nil || hostIP == "" {
		hostIP = getDefaultHostIP()
	}

	nodePort, err := getTraefikNodePortInference(ctx, c, gatewayNamespace)
	if err == nil && nodePort > 0 {
		return hostIP, nodePort, nil
	}

	gateway := &gatewayapiv1.Gateway{}
	if err := c.Get(ctx, types.NamespacedName{Name: gatewayName, Namespace: gatewayNamespace}, gateway); err == nil {
		for _, listener := range gateway.Spec.Listeners {
			if listener.Protocol == gatewayapiv1.HTTPProtocolType {
				return hostIP, int32(listener.Port), nil
			}
		}
	}

	svc := &corev1.Service{}
	if err := c.Get(ctx, types.NamespacedName{Name: "traefik", Namespace: gatewayNamespace}, svc); err == nil {
		for _, port := range svc.Spec.Ports {
			if port.Port == 80 || port.Port == 443 {
				return hostIP, port.Port, nil
			}
		}
	}

	return hostIP, 80, nil
}

func getTraefikNodePortInference(ctx context.Context, c client.Client, namespace string) (int32, error) {
	svc := &corev1.Service{}
	if err := c.Get(ctx, types.NamespacedName{Name: "traefik", Namespace: namespace}, svc); err != nil {
		return 0, err
	}

	for _, port := range svc.Spec.Ports {
		if port.NodePort != 0 {
			return port.NodePort, nil
		}
	}

	return 0, fmt.Errorf("no NodePort found for traefik service")
}

func getHostAndPortFallbackInference(ctx context.Context, c client.Client, gatewayName, gatewayNamespace string) (string, int32, error) {
	svcPort, err := getTraefikServicePort(ctx, c, gatewayName, gatewayNamespace)
	if err != nil {
		hostIP := getDefaultHostIP()
		return hostIP, 80, nil
	}

	gatewayIP, err := getTraefikServiceIP(ctx, c, gatewayName, gatewayNamespace)
	if err != nil {
		hostIP := getDefaultHostIP()
		return hostIP, svcPort, nil
	}

	return gatewayIP, svcPort, nil
}

func getGatewayHostIPInference(ctx context.Context, c client.Client, gatewayName, namespace string) (string, error) {
	gateway := &gatewayapiv1.Gateway{}
	if err := c.Get(ctx, types.NamespacedName{Name: gatewayName, Namespace: namespace}, gateway); err != nil {
		return "", err
	}

	for _, addr := range gateway.Status.Addresses {
		return string(addr.Value), nil
	}

	return "", fmt.Errorf("no address found for gateway %s/%s", namespace, gatewayName)
}

func getTraefikServiceIP(ctx context.Context, c client.Client, gatewayName, gatewayNamespace string) (string, error) {
	svc := &corev1.Service{}
	err := c.Get(ctx, types.NamespacedName{Name: gatewayName, Namespace: gatewayNamespace}, svc)
	if err != nil {
		return "", err
	}

	if len(svc.Status.LoadBalancer.Ingress) > 0 {
		ingress := svc.Status.LoadBalancer.Ingress[0]
		if ingress.IP != "" {
			return ingress.IP, nil
		}
		if ingress.Hostname != "" {
			return ingress.Hostname, nil
		}
	}

	return "", fmt.Errorf("no external IP found for service %s/%s", gatewayNamespace, gatewayName)
}

func getTraefikServicePort(ctx context.Context, c client.Client, gatewayName, gatewayNamespace string) (int32, error) {
	svc := &corev1.Service{}
	err := c.Get(ctx, types.NamespacedName{Name: gatewayName, Namespace: gatewayNamespace}, svc)
	if err != nil {
		return 0, err
	}

	for _, port := range svc.Spec.Ports {
		if port.Protocol == corev1.ProtocolTCP {
			if port.NodePort != 0 {
				return port.NodePort, nil
			}
			return port.Port, nil
		}
	}

	return 0, fmt.Errorf("no port found for service %s/%s", gatewayNamespace, gatewayName)
}

func getDefaultHostIP() string {
	defaultIP := "127.0.0.1"
	interfaces, err := net.Interfaces()
	if err != nil {
		return defaultIP
	}
	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip != nil && ip.To4() != nil {
				return ip.String()
			}
		}
	}
	return defaultIP
}
