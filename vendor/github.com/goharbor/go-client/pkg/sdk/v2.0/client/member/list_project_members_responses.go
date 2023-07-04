// Code generated by go-swagger; DO NOT EDIT.

package member

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"

	"github.com/goharbor/go-client/pkg/sdk/v2.0/models"
)

// ListProjectMembersReader is a Reader for the ListProjectMembers structure.
type ListProjectMembersReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *ListProjectMembersReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewListProjectMembersOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 400:
		result := NewListProjectMembersBadRequest()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 401:
		result := NewListProjectMembersUnauthorized()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 403:
		result := NewListProjectMembersForbidden()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 404:
		result := NewListProjectMembersNotFound()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 500:
		result := NewListProjectMembersInternalServerError()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	default:
		return nil, runtime.NewAPIError("response status code does not match any response statuses defined for this endpoint in the swagger spec", response, response.Code())
	}
}

// NewListProjectMembersOK creates a ListProjectMembersOK with default headers values
func NewListProjectMembersOK() *ListProjectMembersOK {
	return &ListProjectMembersOK{}
}

/*
ListProjectMembersOK describes a response with status code 200, with default header values.

Get project members successfully.
*/
type ListProjectMembersOK struct {

	/* Link refers to the previous page and next page
	 */
	Link string

	/* The total count of members
	 */
	XTotalCount int64

	Payload []*models.ProjectMemberEntity
}

// IsSuccess returns true when this list project members o k response has a 2xx status code
func (o *ListProjectMembersOK) IsSuccess() bool {
	return true
}

// IsRedirect returns true when this list project members o k response has a 3xx status code
func (o *ListProjectMembersOK) IsRedirect() bool {
	return false
}

// IsClientError returns true when this list project members o k response has a 4xx status code
func (o *ListProjectMembersOK) IsClientError() bool {
	return false
}

// IsServerError returns true when this list project members o k response has a 5xx status code
func (o *ListProjectMembersOK) IsServerError() bool {
	return false
}

// IsCode returns true when this list project members o k response a status code equal to that given
func (o *ListProjectMembersOK) IsCode(code int) bool {
	return code == 200
}

func (o *ListProjectMembersOK) Error() string {
	return fmt.Sprintf("[GET /projects/{project_name_or_id}/members][%d] listProjectMembersOK  %+v", 200, o.Payload)
}

func (o *ListProjectMembersOK) String() string {
	return fmt.Sprintf("[GET /projects/{project_name_or_id}/members][%d] listProjectMembersOK  %+v", 200, o.Payload)
}

func (o *ListProjectMembersOK) GetPayload() []*models.ProjectMemberEntity {
	return o.Payload
}

func (o *ListProjectMembersOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	// hydrates response header Link
	hdrLink := response.GetHeader("Link")

	if hdrLink != "" {
		o.Link = hdrLink
	}

	// hydrates response header X-Total-Count
	hdrXTotalCount := response.GetHeader("X-Total-Count")

	if hdrXTotalCount != "" {
		valxTotalCount, err := swag.ConvertInt64(hdrXTotalCount)
		if err != nil {
			return errors.InvalidType("X-Total-Count", "header", "int64", hdrXTotalCount)
		}
		o.XTotalCount = valxTotalCount
	}

	// response payload
	if err := consumer.Consume(response.Body(), &o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewListProjectMembersBadRequest creates a ListProjectMembersBadRequest with default headers values
func NewListProjectMembersBadRequest() *ListProjectMembersBadRequest {
	return &ListProjectMembersBadRequest{}
}

/*
ListProjectMembersBadRequest describes a response with status code 400, with default header values.

Bad request
*/
type ListProjectMembersBadRequest struct {

	/* The ID of the corresponding request for the response
	 */
	XRequestID string

	Payload *models.Errors
}

// IsSuccess returns true when this list project members bad request response has a 2xx status code
func (o *ListProjectMembersBadRequest) IsSuccess() bool {
	return false
}

// IsRedirect returns true when this list project members bad request response has a 3xx status code
func (o *ListProjectMembersBadRequest) IsRedirect() bool {
	return false
}

// IsClientError returns true when this list project members bad request response has a 4xx status code
func (o *ListProjectMembersBadRequest) IsClientError() bool {
	return true
}

// IsServerError returns true when this list project members bad request response has a 5xx status code
func (o *ListProjectMembersBadRequest) IsServerError() bool {
	return false
}

// IsCode returns true when this list project members bad request response a status code equal to that given
func (o *ListProjectMembersBadRequest) IsCode(code int) bool {
	return code == 400
}

func (o *ListProjectMembersBadRequest) Error() string {
	return fmt.Sprintf("[GET /projects/{project_name_or_id}/members][%d] listProjectMembersBadRequest  %+v", 400, o.Payload)
}

func (o *ListProjectMembersBadRequest) String() string {
	return fmt.Sprintf("[GET /projects/{project_name_or_id}/members][%d] listProjectMembersBadRequest  %+v", 400, o.Payload)
}

func (o *ListProjectMembersBadRequest) GetPayload() *models.Errors {
	return o.Payload
}

func (o *ListProjectMembersBadRequest) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	// hydrates response header X-Request-Id
	hdrXRequestID := response.GetHeader("X-Request-Id")

	if hdrXRequestID != "" {
		o.XRequestID = hdrXRequestID
	}

	o.Payload = new(models.Errors)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewListProjectMembersUnauthorized creates a ListProjectMembersUnauthorized with default headers values
func NewListProjectMembersUnauthorized() *ListProjectMembersUnauthorized {
	return &ListProjectMembersUnauthorized{}
}

/*
ListProjectMembersUnauthorized describes a response with status code 401, with default header values.

Unauthorized
*/
type ListProjectMembersUnauthorized struct {

	/* The ID of the corresponding request for the response
	 */
	XRequestID string

	Payload *models.Errors
}

// IsSuccess returns true when this list project members unauthorized response has a 2xx status code
func (o *ListProjectMembersUnauthorized) IsSuccess() bool {
	return false
}

// IsRedirect returns true when this list project members unauthorized response has a 3xx status code
func (o *ListProjectMembersUnauthorized) IsRedirect() bool {
	return false
}

// IsClientError returns true when this list project members unauthorized response has a 4xx status code
func (o *ListProjectMembersUnauthorized) IsClientError() bool {
	return true
}

// IsServerError returns true when this list project members unauthorized response has a 5xx status code
func (o *ListProjectMembersUnauthorized) IsServerError() bool {
	return false
}

// IsCode returns true when this list project members unauthorized response a status code equal to that given
func (o *ListProjectMembersUnauthorized) IsCode(code int) bool {
	return code == 401
}

func (o *ListProjectMembersUnauthorized) Error() string {
	return fmt.Sprintf("[GET /projects/{project_name_or_id}/members][%d] listProjectMembersUnauthorized  %+v", 401, o.Payload)
}

func (o *ListProjectMembersUnauthorized) String() string {
	return fmt.Sprintf("[GET /projects/{project_name_or_id}/members][%d] listProjectMembersUnauthorized  %+v", 401, o.Payload)
}

func (o *ListProjectMembersUnauthorized) GetPayload() *models.Errors {
	return o.Payload
}

func (o *ListProjectMembersUnauthorized) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	// hydrates response header X-Request-Id
	hdrXRequestID := response.GetHeader("X-Request-Id")

	if hdrXRequestID != "" {
		o.XRequestID = hdrXRequestID
	}

	o.Payload = new(models.Errors)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewListProjectMembersForbidden creates a ListProjectMembersForbidden with default headers values
func NewListProjectMembersForbidden() *ListProjectMembersForbidden {
	return &ListProjectMembersForbidden{}
}

/*
ListProjectMembersForbidden describes a response with status code 403, with default header values.

Forbidden
*/
type ListProjectMembersForbidden struct {

	/* The ID of the corresponding request for the response
	 */
	XRequestID string

	Payload *models.Errors
}

// IsSuccess returns true when this list project members forbidden response has a 2xx status code
func (o *ListProjectMembersForbidden) IsSuccess() bool {
	return false
}

// IsRedirect returns true when this list project members forbidden response has a 3xx status code
func (o *ListProjectMembersForbidden) IsRedirect() bool {
	return false
}

// IsClientError returns true when this list project members forbidden response has a 4xx status code
func (o *ListProjectMembersForbidden) IsClientError() bool {
	return true
}

// IsServerError returns true when this list project members forbidden response has a 5xx status code
func (o *ListProjectMembersForbidden) IsServerError() bool {
	return false
}

// IsCode returns true when this list project members forbidden response a status code equal to that given
func (o *ListProjectMembersForbidden) IsCode(code int) bool {
	return code == 403
}

func (o *ListProjectMembersForbidden) Error() string {
	return fmt.Sprintf("[GET /projects/{project_name_or_id}/members][%d] listProjectMembersForbidden  %+v", 403, o.Payload)
}

func (o *ListProjectMembersForbidden) String() string {
	return fmt.Sprintf("[GET /projects/{project_name_or_id}/members][%d] listProjectMembersForbidden  %+v", 403, o.Payload)
}

func (o *ListProjectMembersForbidden) GetPayload() *models.Errors {
	return o.Payload
}

func (o *ListProjectMembersForbidden) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	// hydrates response header X-Request-Id
	hdrXRequestID := response.GetHeader("X-Request-Id")

	if hdrXRequestID != "" {
		o.XRequestID = hdrXRequestID
	}

	o.Payload = new(models.Errors)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewListProjectMembersNotFound creates a ListProjectMembersNotFound with default headers values
func NewListProjectMembersNotFound() *ListProjectMembersNotFound {
	return &ListProjectMembersNotFound{}
}

/*
ListProjectMembersNotFound describes a response with status code 404, with default header values.

Not found
*/
type ListProjectMembersNotFound struct {

	/* The ID of the corresponding request for the response
	 */
	XRequestID string

	Payload *models.Errors
}

// IsSuccess returns true when this list project members not found response has a 2xx status code
func (o *ListProjectMembersNotFound) IsSuccess() bool {
	return false
}

// IsRedirect returns true when this list project members not found response has a 3xx status code
func (o *ListProjectMembersNotFound) IsRedirect() bool {
	return false
}

// IsClientError returns true when this list project members not found response has a 4xx status code
func (o *ListProjectMembersNotFound) IsClientError() bool {
	return true
}

// IsServerError returns true when this list project members not found response has a 5xx status code
func (o *ListProjectMembersNotFound) IsServerError() bool {
	return false
}

// IsCode returns true when this list project members not found response a status code equal to that given
func (o *ListProjectMembersNotFound) IsCode(code int) bool {
	return code == 404
}

func (o *ListProjectMembersNotFound) Error() string {
	return fmt.Sprintf("[GET /projects/{project_name_or_id}/members][%d] listProjectMembersNotFound  %+v", 404, o.Payload)
}

func (o *ListProjectMembersNotFound) String() string {
	return fmt.Sprintf("[GET /projects/{project_name_or_id}/members][%d] listProjectMembersNotFound  %+v", 404, o.Payload)
}

func (o *ListProjectMembersNotFound) GetPayload() *models.Errors {
	return o.Payload
}

func (o *ListProjectMembersNotFound) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	// hydrates response header X-Request-Id
	hdrXRequestID := response.GetHeader("X-Request-Id")

	if hdrXRequestID != "" {
		o.XRequestID = hdrXRequestID
	}

	o.Payload = new(models.Errors)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewListProjectMembersInternalServerError creates a ListProjectMembersInternalServerError with default headers values
func NewListProjectMembersInternalServerError() *ListProjectMembersInternalServerError {
	return &ListProjectMembersInternalServerError{}
}

/*
ListProjectMembersInternalServerError describes a response with status code 500, with default header values.

Internal server error
*/
type ListProjectMembersInternalServerError struct {

	/* The ID of the corresponding request for the response
	 */
	XRequestID string

	Payload *models.Errors
}

// IsSuccess returns true when this list project members internal server error response has a 2xx status code
func (o *ListProjectMembersInternalServerError) IsSuccess() bool {
	return false
}

// IsRedirect returns true when this list project members internal server error response has a 3xx status code
func (o *ListProjectMembersInternalServerError) IsRedirect() bool {
	return false
}

// IsClientError returns true when this list project members internal server error response has a 4xx status code
func (o *ListProjectMembersInternalServerError) IsClientError() bool {
	return false
}

// IsServerError returns true when this list project members internal server error response has a 5xx status code
func (o *ListProjectMembersInternalServerError) IsServerError() bool {
	return true
}

// IsCode returns true when this list project members internal server error response a status code equal to that given
func (o *ListProjectMembersInternalServerError) IsCode(code int) bool {
	return code == 500
}

func (o *ListProjectMembersInternalServerError) Error() string {
	return fmt.Sprintf("[GET /projects/{project_name_or_id}/members][%d] listProjectMembersInternalServerError  %+v", 500, o.Payload)
}

func (o *ListProjectMembersInternalServerError) String() string {
	return fmt.Sprintf("[GET /projects/{project_name_or_id}/members][%d] listProjectMembersInternalServerError  %+v", 500, o.Payload)
}

func (o *ListProjectMembersInternalServerError) GetPayload() *models.Errors {
	return o.Payload
}

func (o *ListProjectMembersInternalServerError) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	// hydrates response header X-Request-Id
	hdrXRequestID := response.GetHeader("X-Request-Id")

	if hdrXRequestID != "" {
		o.XRequestID = hdrXRequestID
	}

	o.Payload = new(models.Errors)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}