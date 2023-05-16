package errors

import (
	"context"
	"net/http"

	"k8s.io/apimachinery/pkg/api/errors"

	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/pkg/util/httputils"
)

// NonCriticalErrors is an array of error statuses, that are non-critical. That means, that this error can be
// silenced and displayed to the user as a warning on the frontend side.
var NonCriticalErrors = []int32{http.StatusForbidden, http.StatusUnauthorized}

func HandleError(err error) ([]error, error) {
	nonCriticalErrors := make([]error, 0)
	return AppendError(err, nonCriticalErrors)
}

func AppendError(err error, nonCriticalErrors []error) ([]error, error) {
	if err != nil {
		if isErrorCritical(err) {
			return nonCriticalErrors, err
		} else {
			log.Warningf("Non-critital error occurred during resource retrieval: %s", err)
		}
	}
	return nonCriticalErrors, nil
}

func isErrorCritical(err error) bool {
	status, ok := err.(*errors.StatusError)
	if !ok {
		return true
	}
	return !contains(NonCriticalErrors, status.ErrStatus.Code)
}

func MergeErrors(errorArrayToMerge ...[]error) (mergedErrors []error) {
	for _, errorArry := range errorArrayToMerge {
		mergedErrors = appendMissing(mergedErrors, errorArry...)
	}
	return
}

func appendMissing(slice []error, toAppend ...error) []error {
	m := make(map[string]bool, 0)

	for _, s := range slice {
		m[s.Error()] = true
	}

	for _, a := range toAppend {
		_, ok := m[a.Error()]
		if !ok {
			slice = append(slice, a)
			m[a.Error()] = true
		}
	}

	return slice
}

func contains(s []int32, e int32) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func NewJSONClientError(err error) *httputils.JSONClientError {
	log.Errorf("Handle error: %#v", err)
	if httpErr, ok := err.(*httputils.JSONClientError); ok {
		return httpErr
	}

	// handle k8s error
	statusCode := http.StatusInternalServerError
	statusError, ok := err.(*errors.StatusError)
	var title string
	var msg string
	if ok && statusError.Status().Code > 0 {
		statusCode = int(statusError.Status().Code)
		title = string(statusError.Status().Reason)
		msg = statusError.Status().Message
	} else {
		return httperrors.NewInternalServerError(err.Error())
	}
	return httputils.NewJsonClientError(statusCode, title, msg)
}

func GeneralServerError(ctx context.Context, w http.ResponseWriter, err error) {
	httperrors.GeneralServerError(ctx, w, NewJSONClientError(err))
}
