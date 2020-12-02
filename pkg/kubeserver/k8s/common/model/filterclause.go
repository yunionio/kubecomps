package model

import (
	"fmt"
	"regexp"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/utils"
)

type SFilterClause struct {
	field    string
	funcName string
	params   []string
}

func QWrap(f func(data *jsonutils.JSONDict) (bool, error), format string, hints ...interface{}) QueryFilter {
	return func(obj IK8sModel) (bool, error) {
		data, err := GetObject(obj)
		msg := fmt.Sprintf(format, hints...)
		if err != nil {
			return false, fmt.Errorf("%s: %v", msg, err)
		}
		ret, err := f(data)
		if err != nil {
			return false, fmt.Errorf("%s: %v", msg, err)
		}
		return ret, nil
	}
}

func QWrapStr(field string, f func(val string) (bool, error), format string, hints ...interface{}) QueryFilter {
	wf := func(obj *jsonutils.JSONDict) (bool, error) {
		val, err := obj.GetString(field)
		if err != nil {
			return false, err
		}
		return f(val)
	}
	return QWrap(wf, format, hints...)
}

type qCond struct{}

func (_ qCond) IN(field string, params []string) QueryFilter {
	f := func(val string) (bool, error) {
		return utils.IsInStringArray(val, params), nil
	}
	return QWrapStr(field, f, "%s.in(%v)", params)
}

func (_ qCond) NOT_IN(field string, params []string) QueryFilter {
	f := func(val string) (bool, error) {
		return utils.IsInStringArray(val, params), nil
	}
	return QWrapStr(field, f, "%s.notin(%v)", params)
}

func (cond qCond) or(op func(val1, val2 string) bool, val string, params ...string) (bool, error) {
	for _, param := range params {
		if op(val, param) {
			return true, nil
		}
	}
	return false, nil
}

func (cond qCond) CONTAINS(field string, params []string) QueryFilter {
	f := func(val string) (bool, error) {
		return cond.or(func(v1, v2 string) bool {
			return strings.Contains(v1, v2)
		}, val, params...)
	}
	return QWrapStr(field, f, "%s.contains(%v)", params)
}

func (cond qCond) LIKE(field string, params []string) QueryFilter {
	return cond.CONTAINS(field, params)
}

func (cond qCond) STARTWITH(field string, params []string) QueryFilter {
	f := func(val string) (bool, error) {
		return cond.or(func(v1, v2 string) bool {
			return strings.HasPrefix(v1, v2)
		}, val, params...)
	}
	return QWrapStr(field, f, "%s.startwith(%v)", params)
}

func (cond qCond) ENDWITH(field string, params []string) QueryFilter {
	f := func(val string) (bool, error) {
		return cond.or(func(v1, v2 string) bool {
			return strings.HasSuffix(v1, v2)
		}, val, params...)
	}
	return QWrapStr(field, f, "%s.endwith(%v)", params)
}

func (cond qCond) EQUALS(field string, params []string) QueryFilter {
	f := func(val string) (bool, error) {
		return cond.or(func(v1, v2 string) bool {
			return strings.Compare(v1, v2) == 0
		}, val, params...)
	}
	return QWrapStr(field, f, "%s.equals(%v)", params)
}

func (cond qCond) NOT_EQUALS(field string, params []string) QueryFilter {
	f := func(val string) (bool, error) {
		return cond.or(func(v1, v2 string) bool {
			return strings.Compare(v1, v2) != 0
		}, val, params...)
	}
	return QWrapStr(field, f, "%s.not_equals(%v)", params)
}

func (fc *SFilterClause) QueryFilter() QueryFilter {
	field := fc.field
	cond := new(qCond)
	switch fc.funcName {
	case "in":
		return cond.IN(field, fc.params)
	case "notin":
		return cond.NOT_IN(field, fc.params)
	/*case "between":
		if len(fc.params) == 2 {
			return sqlchemy.Between(field, fc.params[0], fc.params[1])
		}
	case "ge":
		if len(fc.params) == 1 {
			return sqlchemy.GE(field, fc.params[0])
		}
	case "gt":
		if len(fc.params) == 1 {
			return sqlchemy.GT(field, fc.params[0])
		}
	case "le":
		if len(fc.params) == 1 {
			return sqlchemy.LE(field, fc.params[0])
		}
	case "lt":
		if len(fc.params) == 1 {
			return sqlchemysqlchemy.LT(field, fc.params[0])
		}*/
	case "like":
		return cond.LIKE(field, fc.params)
	case "contains":
		return cond.CONTAINS(field, fc.params)
	case "startswith":
		return cond.STARTWITH(field, fc.params)
	case "endswith":
		return cond.ENDWITH(field, fc.params)
	case "equals":
		return cond.EQUALS(field, fc.params)
	case "notequals":
		if len(fc.params) == 1 {
			return cond.NOT_EQUALS(field, fc.params)
		}
		/*case "isnull":
			return sqlchemy.IsNull(field)
		case "isnotnull":
			return sqlchemy.IsNotNull(field)
		case "isempty":
			return sqlchemy.IsEmpty(field)
		case "isnotempty":
			return sqlchemy.IsNotEmpty(field)
		case "isnullorempty":
			return sqlchemy.IsNullOrEmpty(field)*/
	}
	return nil
}

func (fc *SFilterClause) GetField() string {
	return fc.field
}

func (fc *SFilterClause) String() string {
	return fmt.Sprintf("%s.%s(%s)", fc.field, fc.funcName, strings.Join(fc.params, ","))
}

var (
	filterClausePattern *regexp.Regexp
)

func init() {
	filterClausePattern = regexp.MustCompile(`^(\w+)\.(\w+)\((.*)\)`)
}

func ParseFilterClause(filter string) *SFilterClause {
	matches := filterClausePattern.FindStringSubmatch(filter)
	if matches == nil {
		return nil
	}
	params := utils.FindWords([]byte(matches[3]), 0)
	fc := SFilterClause{field: matches[1], funcName: matches[2], params: params}
	return &fc
}
