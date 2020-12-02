package dataselect

import (
	"fmt"
	"reflect"

	"helm.sh/helm/v3/pkg/release"
	"k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
	tapi "yunion.io/x/kubecomps/pkg/kubeserver/types/apis"
)

type IList interface {
	Append(obj interface{})
	SetMeta(meta tapi.ListMeta)
}

type ListMeta struct {
	tapi.ListMeta
}

func NewListMeta() *ListMeta {
	return &ListMeta{}
}

func (l *ListMeta) SetMeta(meta tapi.ListMeta) {
	l.ListMeta = meta
}

type convertF func(item interface{}) (DataCell, error)

func ToCells(data interface{}, cf convertF) ([]DataCell, error) {
	v := reflect.ValueOf(data)
	if v.Kind() != reflect.Slice {
		return nil, fmt.Errorf("Can't traverse non-slice value, kind: %v", v.Kind())
	}
	cells := make([]DataCell, 0)
	for i := 0; i < v.Len(); i++ {
		cell, err := cf(v.Index(i).Interface())
		if err != nil {
			return nil, err
		}
		cells = append(cells, cell)
	}
	return cells, nil
}

func FromCells(cells []DataCell, list IList) {
	for _, cell := range cells {
		list.Append(cell.GetObject())
	}
}

func ToResourceList(list IList, data interface{}, cellConvertF convertF, dsQuery *DataSelectQuery) error {

	cells, err := ToCells(data, cellConvertF)
	if err != nil {
		return err
	}
	selector := GenericDataSelector(cells, dsQuery)
	FromCells(selector.Data(), list)
	list.SetMeta(selector.ListMeta())
	return nil
}

func getObjectMeta(obj interface{}) (metaV1.ObjectMeta, error) {
	v := reflect.ValueOf(obj)
	f := reflect.Indirect(v).FieldByName("ObjectMeta")
	if !f.IsValid() {
		return metaV1.ObjectMeta{}, fmt.Errorf("Object %#v not have ObjectMeta field", obj)
	}
	meta, ok := f.Interface().(metaV1.ObjectMeta)
	if !ok {
		apiMeta := f.Interface().(api.ObjectMeta)
		meta = metaV1.ObjectMeta{
			Name:              apiMeta.Name,
			Namespace:         apiMeta.Namespace,
			Labels:            apiMeta.Labels,
			Annotations:       apiMeta.Annotations,
			CreationTimestamp: apiMeta.CreationTimestamp,
		}
	}
	return meta, nil
}

func getObjectPodStatus(obj interface{}) (v1.PodStatus, error) {
	v := reflect.ValueOf(obj)
	f := reflect.Indirect(v).FieldByName("Status")
	if !f.IsValid() {
		return v1.PodStatus{}, fmt.Errorf("Object %#v not have Status field", obj)
	}
	status := f.Interface().(v1.PodStatus)
	return status, nil
}

func NewResourceDataCell(obj interface{}) (DataCell, error) {
	meta, err := getObjectMeta(obj)
	if err != nil {
		return ResourceDataCell{}, err
	}
	return ResourceDataCell{ObjectMeta: meta, Object: obj}, nil
}

type ResourceDataCell struct {
	ObjectMeta metaV1.ObjectMeta
	Object     interface{}
}

func (cell ResourceDataCell) GetObject() interface{} {
	return cell.Object
}

func (cell ResourceDataCell) GetProperty(name PropertyName) ComparableValue {
	switch name {
	case NameProperty:
		return StdComparableString(cell.ObjectMeta.Name)
	case CreationTimestampProperty:
		return StdComparableTime(cell.ObjectMeta.CreationTimestamp.Time)
	default:
		return nil
	}
}

func NewNamespaceDataCell(obj interface{}) (DataCell, error) {
	meta, err := getObjectMeta(obj)
	if err != nil {
		return NamespaceDataCell{}, err
	}
	return NamespaceDataCell{ResourceDataCell{meta, obj}}, nil
}

type NamespaceDataCell struct {
	ResourceDataCell
}

func (cell NamespaceDataCell) GetProperty(name PropertyName) ComparableValue {
	switch name {
	case NamespaceProperty:
		return StdComparableEqualString{StdComparableString(cell.ObjectMeta.Namespace)}
	default:
		return cell.ResourceDataCell.GetProperty(name)
	}
}

func NewNamespacePodStatusDataCell(obj interface{}) (DataCell, error) {
	cell, err := NewNamespaceDataCell(obj)
	if err != nil {
		return NamespacePodStatusDataCell{}, err
	}
	status, err := getObjectPodStatus(obj)
	if err != nil {
		return NamespacePodStatusDataCell{}, err
	}
	return NamespacePodStatusDataCell{
		NamespaceDataCell: cell.(NamespaceDataCell),
		Status:            status,
	}, nil
}

type NamespacePodStatusDataCell struct {
	NamespaceDataCell
	Status v1.PodStatus
}

func (cell NamespacePodStatusDataCell) GetProperty(name PropertyName) ComparableValue {
	switch name {
	case StatusProperty:
		return StdComparableString(cell.Status.Phase)
	default:
		return cell.NamespaceDataCell.GetProperty(name)
	}
}

type HelmReleaseDataCell struct {
	Release *release.Release
}

func NewHelmReleaseDataCell(obj interface{}) (DataCell, error) {
	rls, ok := obj.(*release.Release)
	if !ok {
		return nil, fmt.Errorf("Object %#v not *release.Release", obj)
	}
	return HelmReleaseDataCell{Release: rls}, nil
}

func (cell HelmReleaseDataCell) GetObject() interface{} {
	return cell.Release
}

func (cell HelmReleaseDataCell) GetProperty(name PropertyName) ComparableValue {
	switch name {
	case NameProperty:
		return StdComparableString(cell.Release.Name)
	//case CreationTimestampProperty:
	//return StdComparableTime(cell.ObjectMeta.CreationTimestamp.Time)
	case NamespaceProperty:
		return StdComparableString(cell.Release.Namespace)
	default:
		return nil
	}
}

type ChartDataCell struct {
	Chart *api.ChartResult
}

func NewChartDataCell(obj interface{}) (DataCell, error) {
	chart, ok := obj.(*api.ChartResult)
	if !ok {
		return nil, fmt.Errorf("Object %#v not *data.ChartResult", obj)
	}
	return ChartDataCell{Chart: chart}, nil
}

func (cell ChartDataCell) GetObject() interface{} {
	return cell.Chart
}

func (cell ChartDataCell) GetProperty(name PropertyName) ComparableValue {
	switch name {
	case NameProperty:
		return StdComparableString(cell.Chart.ChartVersion.Name)
	default:
		return nil
	}
}

type SecretDataCell struct {
	NamespaceDataCell
	Type v1.SecretType
}

func getSecretType(obj interface{}) (v1.SecretType, error) {
	v := reflect.ValueOf(obj)
	f := reflect.Indirect(v).FieldByName("Type")
	if !f.IsValid() {
		return "", fmt.Errorf("Object %#v not hava Type field", obj)
	}
	return f.Interface().(v1.SecretType), nil
}

func NewSecretDataCell(obj interface{}) (DataCell, error) {
	cell, err := NewNamespaceDataCell(obj)
	if err != nil {
		return SecretDataCell{}, err
	}
	secType, err := getSecretType(obj)
	if err != nil {
		return SecretDataCell{}, err
	}
	return SecretDataCell{
		NamespaceDataCell: cell.(NamespaceDataCell),
		Type:              secType,
	}, nil
}

func (cell SecretDataCell) GetProperty(name PropertyName) ComparableValue {
	switch name {
	case SecretTypeProperty:
		return StdComparableString(string(cell.Type))
	default:
		return cell.NamespaceDataCell.GetProperty(name)
	}
}

type PVCDataCell struct {
	NamespaceDataCell
	Unused string // "true" or "false"
}

func getPVCMountedBy(obj interface{}) ([]string, error) {
	v := reflect.ValueOf(obj)
	f := reflect.Indirect(v).FieldByName("MountedBy")
	if !f.IsValid() {
		return nil, fmt.Errorf("Object %#v not hava MountedBy field", obj)
	}
	return f.Interface().([]string), nil
}

func NewPVCDataCell(obj interface{}) (DataCell, error) {
	cell, err := NewNamespaceDataCell(obj)
	if err != nil {
		return PVCDataCell{}, err
	}
	mountedBy, err := getPVCMountedBy(obj)
	if err != nil {
		return PVCDataCell{}, err
	}
	unused := "false"
	if len(mountedBy) == 0 {
		unused = "true"
	}
	return PVCDataCell{
		NamespaceDataCell: cell.(NamespaceDataCell),
		Unused:            unused,
	}, nil
}

func (cell PVCDataCell) GetProperty(name PropertyName) ComparableValue {
	switch name {
	case PVCUnusedProperty:
		return StdComparableString(cell.Unused)
	default:
		return cell.NamespaceDataCell.GetProperty(name)
	}
}

type ServiceDataCell struct {
	NamespaceDataCell
	Type v1.ServiceType
}

func getServiceSpec(obj interface{}) (v1.ServiceSpec, error) {
	v := reflect.ValueOf(obj)
	f := reflect.Indirect(v).FieldByName("Spec")
	if !f.IsValid() {
		return v1.ServiceSpec{}, fmt.Errorf("Object %#v not hava Spec field", obj)
	}
	return f.Interface().(v1.ServiceSpec), nil
}

func NewServiceDataCell(obj interface{}) (DataCell, error) {
	cell, err := NewNamespaceDataCell(obj)
	if err != nil {
		return ServiceDataCell{}, err
	}
	spec, err := getServiceSpec(obj)
	if err != nil {
		return ServiceDataCell{}, err
	}
	return ServiceDataCell{
		NamespaceDataCell: cell.(NamespaceDataCell),
		Type:              spec.Type,
	}, nil
}

func (cell ServiceDataCell) GetProperty(name PropertyName) ComparableValue {
	switch name {
	case SecretTypeProperty:
		return StdComparableString(string(cell.Type))
	default:
		return cell.NamespaceDataCell.GetProperty(name)
	}
}
