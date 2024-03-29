// Code generated by go-swagger; DO NOT EDIT.

package products

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"
	"net/http"
	"time"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/runtime"
	cr "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
)

// NewDeleteChartrepoRepoChartsNameVersionLabelsIDParams creates a new DeleteChartrepoRepoChartsNameVersionLabelsIDParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewDeleteChartrepoRepoChartsNameVersionLabelsIDParams() *DeleteChartrepoRepoChartsNameVersionLabelsIDParams {
	return &DeleteChartrepoRepoChartsNameVersionLabelsIDParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewDeleteChartrepoRepoChartsNameVersionLabelsIDParamsWithTimeout creates a new DeleteChartrepoRepoChartsNameVersionLabelsIDParams object
// with the ability to set a timeout on a request.
func NewDeleteChartrepoRepoChartsNameVersionLabelsIDParamsWithTimeout(timeout time.Duration) *DeleteChartrepoRepoChartsNameVersionLabelsIDParams {
	return &DeleteChartrepoRepoChartsNameVersionLabelsIDParams{
		timeout: timeout,
	}
}

// NewDeleteChartrepoRepoChartsNameVersionLabelsIDParamsWithContext creates a new DeleteChartrepoRepoChartsNameVersionLabelsIDParams object
// with the ability to set a context for a request.
func NewDeleteChartrepoRepoChartsNameVersionLabelsIDParamsWithContext(ctx context.Context) *DeleteChartrepoRepoChartsNameVersionLabelsIDParams {
	return &DeleteChartrepoRepoChartsNameVersionLabelsIDParams{
		Context: ctx,
	}
}

// NewDeleteChartrepoRepoChartsNameVersionLabelsIDParamsWithHTTPClient creates a new DeleteChartrepoRepoChartsNameVersionLabelsIDParams object
// with the ability to set a custom HTTPClient for a request.
func NewDeleteChartrepoRepoChartsNameVersionLabelsIDParamsWithHTTPClient(client *http.Client) *DeleteChartrepoRepoChartsNameVersionLabelsIDParams {
	return &DeleteChartrepoRepoChartsNameVersionLabelsIDParams{
		HTTPClient: client,
	}
}

/*
DeleteChartrepoRepoChartsNameVersionLabelsIDParams contains all the parameters to send to the API endpoint

	for the delete chartrepo repo charts name version labels ID operation.

	Typically these are written to a http.Request.
*/
type DeleteChartrepoRepoChartsNameVersionLabelsIDParams struct {

	/* ID.

	   The label ID
	*/
	ID int64

	/* Name.

	   The chart name
	*/
	Name string

	/* Repo.

	   The project name
	*/
	Repo string

	/* Version.

	   The chart version
	*/
	Version string

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithDefaults hydrates default values in the delete chartrepo repo charts name version labels ID params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *DeleteChartrepoRepoChartsNameVersionLabelsIDParams) WithDefaults() *DeleteChartrepoRepoChartsNameVersionLabelsIDParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the delete chartrepo repo charts name version labels ID params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *DeleteChartrepoRepoChartsNameVersionLabelsIDParams) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the delete chartrepo repo charts name version labels ID params
func (o *DeleteChartrepoRepoChartsNameVersionLabelsIDParams) WithTimeout(timeout time.Duration) *DeleteChartrepoRepoChartsNameVersionLabelsIDParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the delete chartrepo repo charts name version labels ID params
func (o *DeleteChartrepoRepoChartsNameVersionLabelsIDParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the delete chartrepo repo charts name version labels ID params
func (o *DeleteChartrepoRepoChartsNameVersionLabelsIDParams) WithContext(ctx context.Context) *DeleteChartrepoRepoChartsNameVersionLabelsIDParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the delete chartrepo repo charts name version labels ID params
func (o *DeleteChartrepoRepoChartsNameVersionLabelsIDParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the delete chartrepo repo charts name version labels ID params
func (o *DeleteChartrepoRepoChartsNameVersionLabelsIDParams) WithHTTPClient(client *http.Client) *DeleteChartrepoRepoChartsNameVersionLabelsIDParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the delete chartrepo repo charts name version labels ID params
func (o *DeleteChartrepoRepoChartsNameVersionLabelsIDParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithID adds the id to the delete chartrepo repo charts name version labels ID params
func (o *DeleteChartrepoRepoChartsNameVersionLabelsIDParams) WithID(id int64) *DeleteChartrepoRepoChartsNameVersionLabelsIDParams {
	o.SetID(id)
	return o
}

// SetID adds the id to the delete chartrepo repo charts name version labels ID params
func (o *DeleteChartrepoRepoChartsNameVersionLabelsIDParams) SetID(id int64) {
	o.ID = id
}

// WithName adds the name to the delete chartrepo repo charts name version labels ID params
func (o *DeleteChartrepoRepoChartsNameVersionLabelsIDParams) WithName(name string) *DeleteChartrepoRepoChartsNameVersionLabelsIDParams {
	o.SetName(name)
	return o
}

// SetName adds the name to the delete chartrepo repo charts name version labels ID params
func (o *DeleteChartrepoRepoChartsNameVersionLabelsIDParams) SetName(name string) {
	o.Name = name
}

// WithRepo adds the repo to the delete chartrepo repo charts name version labels ID params
func (o *DeleteChartrepoRepoChartsNameVersionLabelsIDParams) WithRepo(repo string) *DeleteChartrepoRepoChartsNameVersionLabelsIDParams {
	o.SetRepo(repo)
	return o
}

// SetRepo adds the repo to the delete chartrepo repo charts name version labels ID params
func (o *DeleteChartrepoRepoChartsNameVersionLabelsIDParams) SetRepo(repo string) {
	o.Repo = repo
}

// WithVersion adds the version to the delete chartrepo repo charts name version labels ID params
func (o *DeleteChartrepoRepoChartsNameVersionLabelsIDParams) WithVersion(version string) *DeleteChartrepoRepoChartsNameVersionLabelsIDParams {
	o.SetVersion(version)
	return o
}

// SetVersion adds the version to the delete chartrepo repo charts name version labels ID params
func (o *DeleteChartrepoRepoChartsNameVersionLabelsIDParams) SetVersion(version string) {
	o.Version = version
}

// WriteToRequest writes these params to a swagger request
func (o *DeleteChartrepoRepoChartsNameVersionLabelsIDParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error

	// path param id
	if err := r.SetPathParam("id", swag.FormatInt64(o.ID)); err != nil {
		return err
	}

	// path param name
	if err := r.SetPathParam("name", o.Name); err != nil {
		return err
	}

	// path param repo
	if err := r.SetPathParam("repo", o.Repo); err != nil {
		return err
	}

	// path param version
	if err := r.SetPathParam("version", o.Version); err != nil {
		return err
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}
