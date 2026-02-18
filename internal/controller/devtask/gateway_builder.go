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
	"fmt"
	"net"
	"os/exec"
	"strings"

	kaiiov1alpha1 "github.com/hwk114/kai/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gatewayapiv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func NeedsExternalRouting(task *kaiiov1alpha1.DevTask) bool {
	if task.Spec.CodeServer != nil {
		if task.Spec.CodeServer.Routing == nil {
			return true
		}
		return task.Spec.CodeServer.Routing.Enabled
	}
	if task.Spec.Jupyter != nil {
		if task.Spec.Jupyter.Routing == nil {
			return true
		}
		return task.Spec.Jupyter.Routing.Enabled
	}
	if task.Spec.NoVNC != nil {
		if task.Spec.NoVNC.Routing == nil {
			return true
		}
		return task.Spec.NoVNC.Routing.Enabled
	}
	return false
}

func GetGatewayName(task *kaiiov1alpha1.DevTask) string {
	cfg := GetRoutingConfig(task)
	if cfg != nil && cfg.GatewayName != "" {
		return cfg.GatewayName
	}
	return "traefik-gateway"
}

func GetGatewayNamespace(task *kaiiov1alpha1.DevTask) string {
	cfg := GetRoutingConfig(task)
	if cfg != nil && cfg.GatewayNamespace != "" {
		return cfg.GatewayNamespace
	}
	return DefaultGatewayNamespace
}

func GetRoutingConfig(task *kaiiov1alpha1.DevTask) *kaiiov1alpha1.RoutingSpec {
	if task.Spec.CodeServer != nil && task.Spec.CodeServer.Routing != nil {
		return task.Spec.CodeServer.Routing
	}
	if task.Spec.Jupyter != nil && task.Spec.Jupyter.Routing != nil {
		return task.Spec.Jupyter.Routing
	}
	if task.Spec.NoVNC != nil && task.Spec.NoVNC.Routing != nil {
		return task.Spec.NoVNC.Routing
	}
	return nil
}

func GetRoutePath(task *kaiiov1alpha1.DevTask) string {
	cfg := GetRoutingConfig(task)
	if cfg != nil && cfg.PathPrefix != "" {
		return cfg.PathPrefix
	}
	return fmt.Sprintf("/%s/%s/", GetFramework(task), task.Name)
}

func GetHostnames(task *kaiiov1alpha1.DevTask) []gatewayapiv1.Hostname {
	cfg := GetRoutingConfig(task)
	if cfg != nil && len(cfg.Hostnames) > 0 {
		var hostnames []gatewayapiv1.Hostname
		for _, h := range cfg.Hostnames {
			hostnames = append(hostnames, gatewayapiv1.Hostname(h))
		}
		return hostnames
	}
	return nil
}

func GetGatewayAddressAndPort(ctx context.Context, c client.Client, task *kaiiov1alpha1.DevTask) (string, int32, error) {
	gatewayName := GetGatewayName(task)
	gatewayNamespace, _, _ := strings.Cut(GetGatewayNamespace(task), "/")

	nodePort, err := getTraefikNodePort(ctx, c, gatewayNamespace)
	if err == nil && nodePort > 0 {
		hostIP, _ := getGatewayHostIP(ctx, c, gatewayName, gatewayNamespace)
		if hostIP == "" {
			hostIP, _ = GetColimaHostIP()
		}
		return hostIP, nodePort, nil
	}

	hostIP, err := getGatewayHostIP(ctx, c, gatewayName, gatewayNamespace)
	if err == nil && hostIP != "" {
		gateway := &gatewayapiv1.Gateway{}
		if err := c.Get(ctx, types.NamespacedName{Name: gatewayName, Namespace: gatewayNamespace}, gateway); err == nil {
			for _, listener := range gateway.Spec.Listeners {
				if listener.Protocol == gatewayapiv1.HTTPProtocolType {
					return hostIP, listener.Port, nil
				}
			}
		}
	}

	return getHostAndPortFallback(ctx, c, gatewayNamespace)
}

func getTraefikNodePort(ctx context.Context, c client.Client, namespace string) (int32, error) {
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

func getHostAndPortFallback(ctx context.Context, c client.Client, namespace string) (string, int32, error) {
	svcPort, err := getTraefikServicePort(ctx, c, "traefik", namespace)
	if err != nil {
		hostIP, err := GetColimaHostIP()
		if err != nil {
			return "", 0, fmt.Errorf("failed to get host IP: %v", err)
		}
		return hostIP, 80, nil
	}

	gatewayIP, err := getTraefikServiceIP(ctx, c, "traefik", namespace)
	if err != nil {
		hostIP, err := GetColimaHostIP()
		if err != nil {
			return "", 0, fmt.Errorf("failed to get host IP: %v", err)
		}
		return hostIP, svcPort, nil
	}

	return gatewayIP, svcPort, nil
}

func getGatewayHostIP(ctx context.Context, c client.Client, gatewayName, namespace string) (string, error) {
	gateway := &gatewayapiv1.Gateway{}
	if err := c.Get(ctx, types.NamespacedName{Name: gatewayName, Namespace: namespace}, gateway); err != nil {
		return "", err
	}

	for _, addr := range gateway.Status.Addresses {
		return string(addr.Value), nil
	}

	return "", fmt.Errorf("no address found for gateway %s/%s", namespace, gatewayName)
}

func getTraefikServiceIP(ctx context.Context, c client.Client, svcName, namespace string) (string, error) {
	svc := &corev1.Service{}
	err := c.Get(ctx, types.NamespacedName{Name: svcName, Namespace: namespace}, svc)
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

	return "", fmt.Errorf("no external IP found for service %s/%s", namespace, svcName)
}

func getTraefikServicePort(ctx context.Context, c client.Client, svcName, namespace string) (int32, error) {
	svc := &corev1.Service{}
	err := c.Get(ctx, types.NamespacedName{Name: svcName, Namespace: namespace}, svc)
	if err != nil {
		return 0, err
	}

	for _, port := range svc.Spec.Ports {
		if port.Protocol == corev1.ProtocolTCP {
			if port.NodePort != 0 {
				return port.NodePort, nil
			}
		}
	}

	for _, port := range svc.Spec.Ports {
		if port.Protocol == corev1.ProtocolTCP {
			return port.Port, nil
		}
	}

	return 0, fmt.Errorf("no port found for service %s/%s", namespace, svcName)
}

func GetColimaHostIP() (string, error) {
	cmd := exec.Command("colima", "list", "-j")
	output, err := cmd.Output()
	if err != nil {
		return GetDefaultHostIP(), nil
	}

	outputStr := string(output)
	if strings.Contains(outputStr, "\"IP\":") {
		lines := strings.Split(outputStr, "\n")
		for _, line := range lines {
			if strings.Contains(line, "\"IP\"") {
				parts := strings.Split(line, ":")
				if len(parts) >= 2 {
					ip := strings.Trim(strings.Split(parts[1], ",")[0], " \"")
					if net.ParseIP(ip) != nil {
						return ip, nil
					}
				}
			}
		}
	}

	return GetDefaultHostIP(), nil
}

func GetDefaultHostIP() string {
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

func GetExternalURL(task *kaiiov1alpha1.DevTask) string {
	if !NeedsExternalRouting(task) {
		return ""
	}
	cfg := GetRoutingConfig(task)

	scheme := "http"
	path := GetRoutePath(task)

	if cfg != nil && len(cfg.Hostnames) > 0 {
		host := cfg.Hostnames[0]
		port := cfg.ExternalPort
		if cfg.ExternalPort > 0 {
			port = cfg.ExternalPort
		}
		if port == 443 {
			scheme = "https"
		}
		if port == 0 || port == 80 {
			return fmt.Sprintf("%s://%s%s", scheme, host, path)
		}
		return fmt.Sprintf("%s://%s:%d%s", scheme, host, port, path)
	}

	if cfg != nil && cfg.ExternalHost != "" {
		host := cfg.ExternalHost
		port := cfg.ExternalPort
		if port == 0 {
			port = 80
		}
		if port == 443 {
			scheme = "https"
		}
		if port == 80 || port == 443 {
			return fmt.Sprintf("%s://%s%s", scheme, host, path)
		}
		return fmt.Sprintf("%s://%s:%d%s", scheme, host, port, path)
	}

	gatewayName := DefaultGatewayName
	gatewayNamespace := DefaultGatewayNamespace
	if cfg != nil && cfg.GatewayName != "" {
		gatewayName = cfg.GatewayName
	}
	if cfg != nil && cfg.GatewayNamespace != "" {
		gatewayNamespace = cfg.GatewayNamespace
	}

	host := fmt.Sprintf("%s.%s.svc", gatewayName, gatewayNamespace)
	port := int32(80)
	if cfg != nil && cfg.GatewayPort > 0 {
		port = cfg.GatewayPort
	}
	if port == 80 {
		return fmt.Sprintf("%s://%s%s", scheme, host, path)
	}
	return fmt.Sprintf("%s://%s:%d%s", scheme, host, port, path)
}

func GetExternalURLWithGatewayIP(ctx context.Context, c client.Client, task *kaiiov1alpha1.DevTask) (string, error) {
	if !NeedsExternalRouting(task) {
		return "", nil
	}
	cfg := GetRoutingConfig(task)

	scheme := "http"
	path := GetRoutePath(task)

	if cfg != nil && len(cfg.Hostnames) > 0 {
		host := cfg.Hostnames[0]
		port := cfg.ExternalPort
		if cfg.ExternalPort > 0 {
			port = cfg.ExternalPort
		}
		if port == 443 {
			scheme = "https"
		}
		if port == 0 || port == 80 {
			return fmt.Sprintf("%s://%s%s", scheme, host, path), nil
		}
		return fmt.Sprintf("%s://%s:%d%s", scheme, host, port, path), nil
	}

	if cfg != nil && cfg.ExternalHost != "" {
		host := cfg.ExternalHost
		port := cfg.ExternalPort
		if port == 0 {
			port = 80
		}
		if port == 443 {
			scheme = "https"
		}
		if port == 80 || port == 443 {
			return fmt.Sprintf("%s://%s%s", scheme, host, path), nil
		}
		return fmt.Sprintf("%s://%s:%d%s", scheme, host, port, path), nil
	}

	gatewayIP, port, err := GetGatewayAddressAndPort(ctx, c, task)
	if err != nil {
		return "", err
	}

	if port == 80 {
		return fmt.Sprintf("%s://%s%s", scheme, gatewayIP, path), nil
	}
	return fmt.Sprintf("%s://%s:%d%s", scheme, gatewayIP, port, path), nil
}
