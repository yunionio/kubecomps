package dataselect

type DataSelectQuery struct {
	SortQuery   *SortQuery
	FilterQuery *FilterQuery
	LimitQuery  *LimitQuery
	OffsetQuery *OffsetQuery
}

type SortQuery struct {
	SortByList []SortBy
}

type SortBy struct {
	Property  PropertyName
	Ascending bool
}

func NoSort() *SortQuery {
	return &SortQuery{
		SortByList: []SortBy{},
	}
}

type FilterQuery struct {
	FilterByList []FilterBy
}

type FilterBy struct {
	Property PropertyName
	Value    ComparableValue
}

func NoFilter() *FilterQuery {
	return &FilterQuery{
		FilterByList: []FilterBy{},
	}
}

type LimitQuery struct {
	Limit int
}

func NoLimiter() *LimitQuery {
	return &LimitQuery{
		Limit: -1,
	}
}

type OffsetQuery struct {
	Offset int
}

func NoOffset() *OffsetQuery {
	return &OffsetQuery{
		Offset: 0,
	}
}

func NoDataSelect() *DataSelectQuery {
	return NewDataSelectQuery(NoSort(), NoFilter(), NoLimiter(), NoOffset())
}

func NewDataSelectQuery(
	sortQuery *SortQuery,
	filterQuery *FilterQuery,
	limitQuery *LimitQuery,
	offsetQuery *OffsetQuery,
) *DataSelectQuery {
	return &DataSelectQuery{
		SortQuery:   sortQuery,
		FilterQuery: filterQuery,
		LimitQuery:  limitQuery,
		OffsetQuery: offsetQuery,
	}
}

// NewSortQuery takes raw sort options list and returns SortQuery object. For example:
// ["a", "parameter1", "d", "parameter2"] - means that the data should be sorted by
// parameter1 (Ascending) and lster - for results that return equal under parameter1 sort - by parameter2 (Descending)
func NewSortQuery(sortByListRaw []string) *SortQuery {
	if sortByListRaw == nil || len(sortByListRaw)%2 == 1 {
		// Empty sort list or invalid (odd) length
		return NoSort()
	}
	sortByList := []SortBy{}
	for i := 0; i+1 < len(sortByListRaw); i += 2 {
		var ascending bool
		orderOption := sortByListRaw[i]
		if orderOption == "a" {
			ascending = true
		} else if orderOption == "d" {
			ascending = false
		} else {
			// Invalid order option. Only ascending (a), descending (d) options are supported
			return NoSort()
		}

		// parse property name
		propertyName := sortByListRaw[i+1]
		sortBy := SortBy{
			Property:  PropertyName(propertyName),
			Ascending: ascending,
		}
		// Add to the sort options
		sortByList = append(sortByList, sortBy)
	}
	return &SortQuery{
		SortByList: sortByList,
	}
}

// NewFilterQuery takes raw filter options list and returns FilterQuery object. For example:
// ["paramter1", "value1", "parameter2", "value2"] - means that the data should be filtered by
// parameter1 equals value1 and parameter2 equals value2
func NewFilterQuery(filterByListRaw []string) *FilterQuery {
	if filterByListRaw == nil || len(filterByListRaw)%2 == 1 {
		return NoFilter()
	}
	filterByList := []FilterBy{}
	for i := 0; i+1 < len(filterByListRaw); i += 2 {
		propertyName := filterByListRaw[i]
		propertyValue := filterByListRaw[i+1]
		filterBy := NewFilterBy(propertyName, propertyValue)
		filterByList = append(filterByList, filterBy)
	}
	return &FilterQuery{
		FilterByList: filterByList,
	}
}

func NewFilterBy(property, value string) FilterBy {
	return FilterBy{
		Property: PropertyName(property),
		Value:    StdComparableString(value),
	}
}

func (f *FilterQuery) Append(filters ...FilterBy) *FilterQuery {
	f.FilterByList = append(f.FilterByList, filters...)
	return f
}

func NewLimitQuery(limit int) *LimitQuery {
	return &LimitQuery{limit}
}

func NewOffsetQuery(offset int) *OffsetQuery {
	return &OffsetQuery{offset}
}

func DefaultDataSelect() *DataSelectQuery {
	return NoDataSelect()
}
