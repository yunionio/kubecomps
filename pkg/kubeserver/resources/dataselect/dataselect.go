package dataselect

import (
	"sort"

	api "yunion.io/x/kubecomps/pkg/kubeserver/types/apis"
)

type DataCell interface {
	GetProperty(PropertyName) ComparableValue
	// Return the origin wrappered data
	GetObject() interface{}
}

type ComparableValue interface {
	// Compares self with other value. Returns 1 if other value is smaller, 0 if they are the same, -1 if other is larger
	Compare(ComparableValue) int
	// Returns true if self value contains or is equal to other value, false otherwise
	Contains(ComparableValue) bool
}

type DataSelector struct {
	Total int
	// GenericDataList hold generic data cells that are being selected
	GenericDataList []DataCell
	// DataSelectQuery holds the instructions for data select
	DataSelectQuery *DataSelectQuery
}

// Implementation of sort.Interface that we can use built-in sort function on (sort.Sort) for sorting Selectable Data

// Len returns the length of data inside SelectableData
func (s DataSelector) Len() int { return len(s.GenericDataList) }

// Swap swaps 2 indices inside SelectableData
func (s DataSelector) Swap(i, j int) {
	s.GenericDataList[i], s.GenericDataList[j] = s.GenericDataList[j], s.GenericDataList[i]
}

// Less compares 2 indices inside SelectableData and returns true if first index is larger
func (s DataSelector) Less(i, j int) bool {
	for _, sortBy := range s.DataSelectQuery.SortQuery.SortByList {
		a := s.GenericDataList[i].GetProperty(sortBy.Property)
		b := s.GenericDataList[j].GetProperty(sortBy.Property)
		if a == nil || b == nil {
			break
		}
		cmp := a.Compare(b)
		if cmp == 0 {
			// values are the same. Just continue to next sortBy
			continue
		} else {
			return (cmp == -1 && sortBy.Ascending) || (cmp == 1 && !sortBy.Ascending)
		}
	}
	return false
}

// Sort sorts the data inside as instructed by DataSelectQuery and returns itself to allow method chaining
func (s *DataSelector) Sort() *DataSelector {
	sort.Sort(*s)
	return s
}

// Filter the data inside as instructed by DataSelectQuery and returns itself to allow mehtod chaining
func (s *DataSelector) Filter() *DataSelector {
	filteredList := []DataCell{}

	for _, c := range s.GenericDataList {
		matches := true
		for _, filterBy := range s.DataSelectQuery.FilterQuery.FilterByList {
			v := c.GetProperty(filterBy.Property)
			if v == nil {
				matches = false
				continue
			}
			if !v.Contains(filterBy.Value) {
				matches = false
				continue
			}
		}
		if matches {
			filteredList = append(filteredList, c)
		}
	}

	s.GenericDataList = filteredList
	s.Total = len(filteredList)
	return s
}

func (s *DataSelector) Limit() *DataSelector {
	limit := s.DataSelectQuery.LimitQuery.Limit
	if limit < 0 {
		// -1 means not do limit query
		return s
	}

	if s.Total > limit {
		if limit <= len(s.GenericDataList) {
			s.GenericDataList = s.GenericDataList[:limit]
		}
	}
	return s
}

func (s *DataSelector) Offset() *DataSelector {
	offset := s.DataSelectQuery.OffsetQuery.Offset
	if offset == 0 {
		return s
	}

	if s.Total > offset {
		s.GenericDataList = s.GenericDataList[offset:]
	} else {
		s.GenericDataList = make([]DataCell, 0)
	}
	return s
}

func (s *DataSelector) ListMeta() api.ListMeta {
	return api.ListMeta{
		Total:  s.Total,
		Limit:  s.DataSelectQuery.LimitQuery.Limit,
		Offset: s.DataSelectQuery.OffsetQuery.Offset,
	}
}

func GenericDataSelector(dataList []DataCell, dsQuery *DataSelectQuery) *DataSelector {
	selectableData := DataSelector{
		Total:           len(dataList),
		GenericDataList: dataList,
		DataSelectQuery: dsQuery,
	}
	return selectableData.Sort().Filter().Offset().Limit()
}

func (s *DataSelector) Data() []DataCell {
	return s.GenericDataList
}
