package models

import (
	"context"
	"fmt"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
	"yunion.io/x/kubecomps/pkg/kubeserver/client"
)

var (
	ingressManager *SIngressManager
)

func init() {
	GetIngressManager()
}

func GetIngressManager() *SIngressManager {
	if ingressManager == nil {
		ingressManager = NewK8sNamespaceModelManager(func() ISyncableManager {
			return &SIngressManager{
				SNamespaceResourceBaseManager: NewNamespaceResourceBaseManager(
					new(SIngress),
					"ingresses_tbl",
					"ingress",
					"ingresses",
					api.ResourceNameIngress,
					"",
					"",
					api.KindNameIngress,
					new(unstructured.Unstructured),
				),
			}
		}).(*SIngressManager)
	}
	return ingressManager
}

type SIngressManager struct {
	SNamespaceResourceBaseManager
}

type SIngress struct {
	SNamespaceResourceBase
}

func (m *SIngressManager) NewRemoteObjectForCreate(model IClusterModel, cli *client.ClusterManager, data jsonutils.JSONObject) (interface{}, error) {
	input := new(api.IngressCreateInputV2)
	res := new(unstructured.Unstructured)
	spec := make(map[string]interface{})
	var (
		err              error
		objMeta          v1.ObjectMeta
		rules            []jsonutils.JSONObject
		tlses            []jsonutils.JSONObject
		backend          jsonutils.JSONObject
		rulesArray       []interface{}
		tlsArray         []interface{}
		ruleResult       map[string]interface{}
		backendResult    map[string]interface{}
		tlsResult        map[string]interface{}
		ingressClassName string
	)

	err = data.Unmarshal(input)
	if err != nil {
		return nil, errors.Wrap(err, "ingress input unmarshal error")
	}
	objMeta, err = input.ToObjectMeta(model.(api.INamespaceGetter))
	if err != nil {
		return nil, errors.Wrap(err, "ingress input get meta error")
	}

	// meta object
	res.SetName(objMeta.Name)
	res.SetNamespace(objMeta.Namespace)
	res.SetLabels(objMeta.Labels)
	anno := objMeta.Annotations
	// optional ingress classname
	ingressClassName, _ = data.GetString("ingressClassName")
	if ingressClassName != "" {
		spec["ingressClassName"] = ingressClassName
	}
	if ingressClassName == "nginx" {
		anno["kubernetes.io/ingress.class"] = ingressClassName
		ssConf := input.StickySession
		if ssConf != nil && ssConf.Enabled {
			anno["nginx.ingress.kubernetes.io/affinity"] = "cookie"
			anno["nginx.ingress.kubernetes.io/session-cookie-expires"] = fmt.Sprintf("%d", ssConf.CookieExpires)
			anno["nginx.ingress.kubernetes.io/session-cookie-max-age"] = fmt.Sprintf("%d", ssConf.CookieExpires)
			anno["nginx.ingress.kubernetes.io/session-cookie-name"] = ssConf.Name
		}
	}
	res.SetAnnotations(anno)
	// optional backend
	backend, err = data.Get("backend")
	if err == nil {
		backendResult, err = m.generateBackendFromJson(backend)
		if err == nil && len(backendResult) > 0 {
			spec["backend"] = backendResult
			spec["defaultBackend"] = backendResult
		}
	}
	// optional TLS
	tlses, err = data.GetArray("tls")
	if err == nil {
		for _, tls := range tlses {
			tlsResult, err = m.generateTLSFromJson(tls)
			if err != nil {
				continue
			}
			tlsArray = append(tlsArray, tlsResult)
		}
		if len(tlsArray) > 0 {
			spec["tls"] = tlsArray
		}
	}
	// Rules
	rules, err = data.GetArray("rules")
	if err == nil {
		for _, rule := range rules {
			ruleResult, err = m.generateRuleFromJson(rule)
			if err != nil {
				log.Warningf("generateRuleFromJson %s: %v", rule.PrettyString(), err)
				continue
			}
			rulesArray = append(rulesArray, ruleResult)
		}
		if len(rulesArray) > 0 {
			spec["rules"] = rulesArray
		}
	}

	err = unstructured.SetNestedMap(res.Object, spec, "spec")
	if err != nil {
		return nil, errors.Wrap(err, "set nested map of unstructured")
	}

	return res, nil
}

func (obj *SIngress) getEndpoints(rObj *unstructured.Unstructured) []api.Endpoint {
	endpoints := make([]api.Endpoint, 0)
	ingress, _, _ := unstructured.NestedSlice(rObj.Object, "status", "loadBalancer", "ingress")
	if len(ingress) > 0 {
		for _, status := range ingress {
			ip, _, _ := unstructured.NestedString(status.(map[string]interface{}), "ip")
			endpoint := api.Endpoint{Host: ip}
			endpoints = append(endpoints, endpoint)
		}
	}
	return endpoints
}

func (obj *SIngress) GetDetails(
	ctx context.Context,
	cli *client.ClusterManager,
	base interface{},
	k8sObj runtime.Object,
	isList bool,
) interface{} {
	ing := k8sObj.(*unstructured.Unstructured)
	detail := api.IngressDetailV2{
		NamespaceResourceDetail: obj.SNamespaceResourceBase.GetDetails(ctx, cli, base, k8sObj, isList).(api.NamespaceResourceDetail),
		Endpoints:               obj.getEndpoints(ing),
	}
	if isList {
		return detail
	}
	detail.Spec, _, _ = unstructured.NestedMap(ing.Object, "spec")
	detail.Status, _, _ = unstructured.NestedMap(ing.Object, "status")
	return detail
}

func (m *SIngressManager) generateRuleFromJson(rule jsonutils.JSONObject) (map[string]interface{}, error) {
	backendResult := make(map[string]interface{})
	var (
		pathsArray []interface{}
		paths      []jsonutils.JSONObject
		backend    jsonutils.JSONObject
		err        error
		host       string
		pathname   string
	)

	paths, err = rule.GetArray("paths")
	if err != nil {
		return nil, errors.Wrap(err, "rule not contain paths")
	}
	host, _ = rule.GetString("host")
	for _, path := range paths {
		pathname, err = path.GetString("path")
		if err != nil {
			return nil, errors.Wrap(err, "path not contain name")
		}
		backend, err = path.Get("backend")
		if err != nil {
			return nil, errors.Wrap(err, "path not contain name")
		}
		backendResult, err = m.generateBackendFromJson(backend)
		if err != nil {
			return nil, errors.Wrap(err, "generate backend from JSON")
		}
		pathsArray = append(pathsArray, map[string]interface{}{
			"path":     pathname,
			"pathType": "Exact",
			"backend":  backendResult,
		})
	}

	return map[string]interface{}{
		"host": host,
		"http": map[string]interface{}{
			"paths": pathsArray,
		},
	}, nil
}

func (m *SIngressManager) generateBackendFromJson(backend jsonutils.JSONObject) (map[string]interface{}, error) {
	var (
		serviceName string
		servicePort int64
		err         error
	)

	serviceName, err = backend.GetString("serviceName")
	if err != nil {
		return nil, errors.Wrap(err, "backend not contain serviceName")
	}
	servicePort, err = backend.Int("servicePort")
	if err != nil {
		return nil, errors.Wrap(err, "backend not contain servicePort")
	}

	return map[string]interface{}{
		"serviceName": serviceName,
		"servicePort": servicePort,
		"service": map[string]interface{}{
			"name": serviceName,
			"port": map[string]interface{}{
				"number": servicePort,
			},
		},
	}, nil
}

func (m *SIngressManager) generateTLSFromJson(tls jsonutils.JSONObject) (map[string]interface{}, error) {
	var (
		secretName string
		hostResult []string
		hosts      []jsonutils.JSONObject
		err        error
	)

	hosts, err = tls.GetArray("hosts")
	if err != nil {
		return nil, errors.Wrap(err, "tls not contain hosts")
	}
	for _, host := range hosts {
		hostResult = append(hostResult, host.String())
	}
	secretName, err = tls.GetString("secretName")
	if err != nil {
		return nil, errors.Wrap(err, "tls not contain secretName")
	}

	return map[string]interface{}{
		"secretName": secretName,
		"hosts":      hostResult,
	}, nil
}
