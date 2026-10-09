package models

import (
	"encoding/json"

	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	"sigs.k8s.io/yaml"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
	"yunion.io/x/kubecomps/pkg/kubeserver/client"
)

func listVolcanoRemoteObjects(cli *client.ClusterManager, kind string, list func(*client.ClusterManager) ([]interface{}, error)) ([]interface{}, error) {
	handler := cli.GetHandler()
	if handler == nil || handler.GetIndexer() == nil || !handler.GetIndexer().HasGenericInformer(kind) {
		return []interface{}{}, nil
	}
	return list(cli)
}

func asUnstructured(obj runtime.Object) (*unstructured.Unstructured, error) {
	if obj == nil {
		return nil, errors.Error("empty k8s object")
	}
	if u, ok := obj.(*unstructured.Unstructured); ok {
		return u, nil
	}
	return nil, errors.Errorf("object %T is not unstructured", obj)
}

func nestedString(u *unstructured.Unstructured, fields ...string) string {
	if u == nil {
		return ""
	}
	v, found, err := unstructured.NestedString(u.Object, fields...)
	if err != nil || !found {
		return ""
	}
	return v
}

func nestedInt64(u *unstructured.Unstructured, fields ...string) int64 {
	if u == nil {
		return 0
	}
	val, found, err := unstructured.NestedFieldNoCopy(u.Object, fields...)
	if err != nil || !found || val == nil {
		return 0
	}
	switch n := val.(type) {
	case int64:
		return n
	case int32:
		return int64(n)
	case float64:
		return int64(n)
	case json.Number:
		i, err := n.Int64()
		if err != nil {
			return 0
		}
		return i
	default:
		return 0
	}
}

func nestedBool(u *unstructured.Unstructured, fields ...string) (bool, bool) {
	if u == nil {
		return false, false
	}
	v, found, err := unstructured.NestedBool(u.Object, fields...)
	if err != nil || !found {
		return false, false
	}
	return v, true
}

func parseVolcanoManifest(raw, expectKind, expectAPIVersion string) (*unstructured.Unstructured, error) {
	if raw == "" {
		return nil, httperrors.NewNotEmptyError("yaml is empty")
	}
	u := &unstructured.Unstructured{}
	if err := yaml.Unmarshal([]byte(raw), &u.Object); err != nil {
		return nil, httperrors.NewInputParameterError("invalid manifest: %v", err)
	}
	if len(u.Object) == 0 {
		return nil, httperrors.NewInputParameterError("manifest is empty")
	}
	if !volcanoKindAccepted(u.GetKind(), expectKind) {
		return nil, httperrors.NewInputParameterError("kind must be %s", expectKind)
	}
	// Informer objects use the internal kind so they do not collide with batch/v1 Job.
	// The cluster API still requires the CRD kind.
	u.SetKind(expectKind)
	if u.GetAPIVersion() != expectAPIVersion {
		return nil, httperrors.NewInputParameterError("apiVersion must be %s", expectAPIVersion)
	}
	if u.GetName() == "" {
		return nil, httperrors.NewNotEmptyError("metadata.name is empty")
	}
	return u, nil
}

func replaceVolcanoObject(res IClusterModel, resourceName, namespace string, data jsonutils.JSONObject, expectKind, expectAPIVersion string) (*unstructured.Unstructured, error) {
	u, err := parseVolcanoManifest(data.String(), expectKind, expectAPIVersion)
	if err != nil {
		return nil, err
	}
	u.SetName(res.GetName())
	if namespace != "" {
		u.SetNamespace(namespace)
	}
	if u.GetResourceVersion() == "" {
		current, err := GetK8sObject(res)
		if err != nil {
			return nil, err
		}
		cur, err := asUnstructured(current)
		if err != nil {
			return nil, err
		}
		u.SetResourceVersion(cur.GetResourceVersion())
	}
	cli, err := res.GetClusterClient()
	if err != nil {
		return nil, err
	}
	updated, err := replaceVolcanoUnstructured(cli, resourceName, u)
	if err != nil {
		return nil, err
	}
	if updated != nil {
		updated.SetKind(expectKind)
	}
	return updated, nil
}

func replaceVolcanoUnstructured(cli *client.ClusterManager, resourceName string, u *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	updated, err := cli.GetHandler().ReplaceUnstructured(resourceName, u)
	if err == nil || !apierrors.IsUnauthorized(err) {
		if err != nil {
			return nil, errors.Wrap(err, "replace volcano object")
		}
		return updated, nil
	}
	reloaded, rerr := replaceVolcanoWithReloadedKubeconfig(cli, resourceName, u)
	if rerr == nil {
		return reloaded, nil
	}
	return nil, httperrors.NewUnauthorizedError(
		"cluster API server rejected the kubeconfig while updating %s %s/%s (%v). Update the cluster kubeconfig and sync the cluster, then save again",
		resourceName, u.GetNamespace(), u.GetName(), err,
	)
}

func replaceVolcanoWithReloadedKubeconfig(cli *client.ClusterManager, resourceName string, u *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	cluster := cli.GetClusterObject()
	if cluster == nil {
		return nil, errors.Error("cluster client has no cluster")
	}
	kubeconfig, err := cluster.GetKubeconfig()
	if err != nil {
		return nil, errors.Wrap(err, "reload kubeconfig")
	}
	apiServer, err := cluster.GetAPIServer()
	if err != nil {
		return nil, errors.Wrap(err, "get api server")
	}
	_, cfg, err := client.BuildClient(apiServer, kubeconfig)
	if err != nil {
		return nil, errors.Wrap(err, "rebuild cluster client")
	}
	dc, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "rebuild dynamic client")
	}
	return cli.GetHandler().ReplaceUnstructuredUsing(dc, resourceName, u)
}

func volcanoKindAccepted(got, expectKind string) bool {
	if got == expectKind {
		return true
	}
	switch expectKind {
	case api.VolcanoJobKind:
		return got == string(api.KindNameVolcanoJob)
	case api.VolcanoQueueKind:
		return got == string(api.KindNameVolcanoQueue)
	case api.VolcanoPodGroupKind:
		return got == string(api.KindNameVolcanoPodGroup)
	case api.VolcanoHyperNodeKind:
		return got == string(api.KindNameVolcanoHyperNode)
	default:
		return false
	}
}

func volcanoRawdata(obj jsonutils.JSONObject, kind string) jsonutils.JSONObject {
	dict, ok := obj.(*jsonutils.JSONDict)
	if !ok || dict == nil {
		return obj
	}
	dict.Set("kind", jsonutils.NewString(kind))
	// API Server records field ownership here. kubectl hides it; the YAML editor shows the raw object.
	if meta, err := dict.Get("metadata"); err == nil {
		if md, ok := meta.(*jsonutils.JSONDict); ok {
			md.Remove("managedFields")
		}
	}
	return dict
}

func boolOr(v *bool, def bool) bool {
	if v == nil {
		return def
	}
	return *v
}

func validateQuantity(name, value string) error {
	if value == "" {
		return nil
	}
	if _, err := resource.ParseQuantity(value); err != nil {
		return httperrors.NewInputParameterError("%s %q is not a quantity: %v", name, value, err)
	}
	return nil
}

func compareQuantity(leftName, left, rightName, right, op string) error {
	if left == "" || right == "" {
		return nil
	}
	lq, err := resource.ParseQuantity(left)
	if err != nil {
		return httperrors.NewInputParameterError("%s %q is not a quantity: %v", leftName, left, err)
	}
	rq, err := resource.ParseQuantity(right)
	if err != nil {
		return httperrors.NewInputParameterError("%s %q is not a quantity: %v", rightName, right, err)
	}
	cmp := lq.Cmp(rq)
	switch op {
	case "gte":
		if cmp < 0 {
			return httperrors.NewInputParameterError("%s must be >= %s", leftName, rightName)
		}
	case "lte":
		if cmp > 0 {
			return httperrors.NewInputParameterError("%s must be <= %s", leftName, rightName)
		}
	}
	return nil
}

func validateQueueSpec(spec *api.VCQueueSpecInput) error {
	if spec.Weight < 1 {
		return httperrors.NewInputParameterError("weight must be >= 1")
	}
	if spec.Reclaimable == nil {
		reclaimable := true
		spec.Reclaimable = &reclaimable
	}
	for _, item := range []struct {
		name  string
		value string
	}{
		{"guaranteeCpu", spec.GuaranteeCpu},
		{"guaranteeMemory", spec.GuaranteeMemory},
		{"capabilityCpu", spec.CapabilityCpu},
		{"capabilityMemory", spec.CapabilityMemory},
		{"deservedCpu", spec.DeservedCpu},
		{"deservedMemory", spec.DeservedMemory},
	} {
		if err := validateQuantity(item.name, item.value); err != nil {
			return err
		}
	}
	if spec.GuaranteeCpu != "" && spec.DeservedCpu == "" {
		return httperrors.NewInputParameterError("deservedCpu is required when guaranteeCpu is set")
	}
	if spec.GuaranteeMemory != "" && spec.DeservedMemory == "" {
		return httperrors.NewInputParameterError("deservedMemory is required when guaranteeMemory is set")
	}
	if err := compareQuantity("deservedCpu", spec.DeservedCpu, "guaranteeCpu", spec.GuaranteeCpu, "gte"); err != nil {
		return err
	}
	if err := compareQuantity("deservedMemory", spec.DeservedMemory, "guaranteeMemory", spec.GuaranteeMemory, "gte"); err != nil {
		return err
	}
	if err := compareQuantity("deservedCpu", spec.DeservedCpu, "capabilityCpu", spec.CapabilityCpu, "lte"); err != nil {
		return err
	}
	if err := compareQuantity("deservedMemory", spec.DeservedMemory, "capabilityMemory", spec.CapabilityMemory, "lte"); err != nil {
		return err
	}
	return nil
}

func applyQueueSpec(u *unstructured.Unstructured, spec *api.VCQueueSpecInput) error {
	if u.Object == nil {
		u.Object = map[string]interface{}{}
	}
	if u.GetAPIVersion() == "" {
		u.SetAPIVersion(api.VolcanoQueueAPIVersion)
	}
	if u.GetKind() == "" {
		u.SetKind(api.VolcanoQueueKind)
	}
	if err := unstructured.SetNestedField(u.Object, int64(spec.Weight), "spec", "weight"); err != nil {
		return err
	}
	if err := unstructured.SetNestedField(u.Object, int64(spec.Priority), "spec", "priority"); err != nil {
		return err
	}
	if err := unstructured.SetNestedField(u.Object, boolOr(spec.Reclaimable, true), "spec", "reclaimable"); err != nil {
		return err
	}
	if spec.Parent != "" {
		if err := unstructured.SetNestedField(u.Object, spec.Parent, "spec", "parent"); err != nil {
			return err
		}
	} else {
		unstructured.RemoveNestedField(u.Object, "spec", "parent")
	}
	if err := setResourceMap(u, resourcePair(spec.GuaranteeCpu, spec.GuaranteeMemory), true, "spec", "guarantee"); err != nil {
		return err
	}
	if err := setResourceMap(u, resourcePair(spec.CapabilityCpu, spec.CapabilityMemory), false, "spec", "capability"); err != nil {
		return err
	}
	if err := setResourceMap(u, resourcePair(spec.DeservedCpu, spec.DeservedMemory), false, "spec", "deserved"); err != nil {
		return err
	}
	return nil
}

func setResourceMap(u *unstructured.Unstructured, pair map[string]interface{}, nested bool, fields ...string) error {
	if pair == nil {
		unstructured.RemoveNestedField(u.Object, fields...)
		return nil
	}
	body := pair
	if nested {
		body = map[string]interface{}{"resource": pair}
	}
	return unstructured.SetNestedMap(u.Object, body, fields...)
}

func resourcePair(cpu, memory string) map[string]interface{} {
	out := map[string]interface{}{}
	if cpu != "" {
		out["cpu"] = cpu
	}
	if memory != "" {
		out["memory"] = memory
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func newQueueObject(name string, spec *api.VCQueueSpecInput) (*unstructured.Unstructured, error) {
	u := &unstructured.Unstructured{}
	u.SetAPIVersion(api.VolcanoQueueAPIVersion)
	u.SetKind(api.VolcanoQueueKind)
	u.SetName(name)
	if err := applyQueueSpec(u, spec); err != nil {
		return nil, err
	}
	return u, nil
}

func queueDetailFromUnstructured(base api.ClusterResourceDetail, u *unstructured.Unstructured) api.VCQueueDetail {
	reclaimable, found := nestedBool(u, "spec", "reclaimable")
	if !found {
		reclaimable = true
	}
	guaranteeCpu, guaranteeMemory := resourcePairFromUnstructured(u, "spec", "guarantee", "resource")
	if guaranteeCpu == "" && guaranteeMemory == "" {
		guaranteeCpu, guaranteeMemory = resourcePairFromUnstructured(u, "spec", "guarantee")
	}
	capabilityCpu, capabilityMemory := resourcePairFromUnstructured(u, "spec", "capability")
	deservedCpu, deservedMemory := resourcePairFromUnstructured(u, "spec", "deserved")
	phase := nestedString(u, "status", "state")
	if phase == "" {
		phase = api.ClusterResourceStatusActive
	}
	return api.VCQueueDetail{
		ClusterResourceDetail: base,
		Parent:                nestedString(u, "spec", "parent"),
		Weight:                nestedInt64(u, "spec", "weight"),
		Priority:              nestedInt64(u, "spec", "priority"),
		Reclaimable:           reclaimable,
		GuaranteeCpu:          guaranteeCpu,
		GuaranteeMemory:       guaranteeMemory,
		CapabilityCpu:         capabilityCpu,
		CapabilityMemory:      capabilityMemory,
		DeservedCpu:           deservedCpu,
		DeservedMemory:        deservedMemory,
		Status:                phase,
	}
}

func resourcePairFromUnstructured(u *unstructured.Unstructured, fields ...string) (string, string) {
	return nestedString(u, append(fields, "cpu")...), nestedString(u, append(fields, "memory")...)
}

func volcanoPhase(u *unstructured.Unstructured, fields ...string) string {
	phase := nestedString(u, fields...)
	if phase == "" {
		return api.ClusterResourceStatusActive
	}
	return phase
}

func isSystemQueue(name string) bool {
	return name == "root" || name == "default"
}

const (
	volcanoJobNameLabel    = "volcano.sh/job-name"
	volcanoQueueNameLabel  = "volcano.sh/queue-name"
	volcanoGroupAnnotation = "scheduling.k8s.io/group-name"
)

func volcanoPodsBySelector(cli *client.ClusterManager, namespace string, selector labels.Selector) ([]*v1.Pod, error) {
	if cli == nil || cli.GetHandler() == nil || cli.GetHandler().GetIndexer() == nil {
		return []*v1.Pod{}, nil
	}
	pods, err := GetPodManager().GetRawPodsBySelector(cli, namespace, selector)
	if err != nil || pods == nil {
		return []*v1.Pod{}, err
	}
	return pods, nil
}

func volcanoPodsByJob(cli *client.ClusterManager, namespace, jobName string) ([]*v1.Pod, error) {
	if jobName == "" {
		return []*v1.Pod{}, nil
	}
	selector := labels.SelectorFromSet(labels.Set{volcanoJobNameLabel: jobName})
	return volcanoPodsBySelector(cli, namespace, selector)
}

func volcanoPodsByQueue(cli *client.ClusterManager, queueName string) ([]*v1.Pod, error) {
	if queueName == "" {
		return []*v1.Pod{}, nil
	}
	selector := labels.SelectorFromSet(labels.Set{volcanoQueueNameLabel: queueName})
	return volcanoPodsBySelector(cli, v1.NamespaceAll, selector)
}

func volcanoPodsByPodGroup(cli *client.ClusterManager, namespace, podGroupName string) ([]*v1.Pod, error) {
	if podGroupName == "" {
		return []*v1.Pod{}, nil
	}
	pods, err := volcanoPodsBySelector(cli, namespace, labels.Everything())
	if err != nil {
		return []*v1.Pod{}, err
	}
	matched := make([]*v1.Pod, 0)
	for _, pod := range pods {
		if pod.Annotations[volcanoGroupAnnotation] == podGroupName {
			matched = append(matched, pod)
		}
	}
	return matched, nil
}

func volcanoPodGroupName(u *unstructured.Unstructured) string {
	if u == nil || u.GetName() == "" || u.GetUID() == "" {
		return ""
	}
	return u.GetName() + "-" + string(u.GetUID())
}

func volcanoJobOwnerName(u *unstructured.Unstructured) string {
	if u == nil {
		return ""
	}
	for _, ref := range u.GetOwnerReferences() {
		if ref.Kind == api.VolcanoJobKind && ref.APIVersion == api.VolcanoJobAPIVersion {
			return ref.Name
		}
	}
	return ""
}
