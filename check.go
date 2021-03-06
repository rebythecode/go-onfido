package onfido

import (
	"bytes"
	"context"
	"encoding/json"
	"time"
)

// CheckType represents a check type (express, standard)
type CheckType string

// CheckStatus represents a status of a check
type CheckStatus string

// CheckResult represents a result of a check (clear, consider)
type CheckResult string

// Supported check types
const (
	CheckTypeExpress  CheckType = "express"
	CheckTypeStandard CheckType = "standard"

	CheckStatusInProgress        CheckStatus = "in_progress"
	CheckStatusAwaitingApplicant CheckStatus = "awaiting_applicant"
	CheckStatusComplete          CheckStatus = "complete"
	CheckStatusWithdrawn         CheckStatus = "withdrawn"
	CheckStatusPaused            CheckStatus = "paused"
	CheckStatusReopened          CheckStatus = "reopened"

	CheckResultClear    CheckResult = "clear"
	CheckResultConsider CheckResult = "consider"
)

// CheckRequest represents a check request to Onfido API
type CheckRequest struct {
	ApplicantID           string       `json:"applicant_id,omitempty"`
	ReportNames           []ReportName `json:"report_names"`
	ApplicantProvidesData bool         `json:"applicant_provides_data,omitempty"`
	Asynchronous          bool         `json:"asynchronous,omitempty"`
	Tags                  []string     `json:"tags,omitempty"`
	SupressFormEmails     bool         `json:"suppress_form_emails,omitempty"`
	RedirectURI           string       `json:"redirect_uri,omitempty"`
	// Consider is used for Sandbox Testing of multiple report scenarios.
	// see https://documentation.onfido.com/#sandbox-responses
	Consider []ReportName `json:"consider,omitempty"`
}

// Check represents a check in Onfido API
type Check struct {
	ID                    string      `json:"id,omitempty"`
	CreatedAt             *time.Time  `json:"created_at,omitempty"`
	Href                  string      `json:"href,omitempty"`
	ApplicantID           string      `json:"applicant_id,omitempty"`
	ApplicantProvidesData bool        `json:"applicant_provides_data,omitempty"`
	Status                CheckStatus `json:"status,omitempty"`
	Result                CheckResult `json:"result,omitempty"`
	FormURI               string      `json:"form_uri,omitempty"`
	RedirectURI           string      `json:"redirect_uri,omitempty"`
	ResultsURI            string      `json:"results_uri,omitempty"`
	Reports               []*Report   `json:"reports,omitempty"`
	Tags                  []string    `json:"tags,omitempty"`
	Paused                bool        `json:"paused,omitempty"`
	Version               string      `json:"version,omitempty"`
}

// CheckRetrieved represents a check in the Onfido API which has been retrieved.
// This is subtly different to the Check type above, as the Reports slice
// is just a string of Report IDs, not fully expanded Report objects.
// See https://documentation.onfido.com/?shell#check-object (Shell)
type CheckRetrieved struct {
	ID                    string      `json:"id,omitempty"`
	CreatedAt             *time.Time  `json:"created_at,omitempty"`
	Href                  string      `json:"href,omitempty"`
	ApplicantID           string      `json:"applicant_id,omitempty"`
	ApplicantProvidesData bool        `json:"applicant_provides_data,omitempty"`
	Status                CheckStatus `json:"status,omitempty"`
	Result                CheckResult `json:"result,omitempty"`
	FormURI               string      `json:"form_uri,omitempty"`
	RedirectURI           string      `json:"redirect_uri,omitempty"`
	ResultsURI            string      `json:"results_uri,omitempty"`
	ReportIDs             []string    `json:"report_ids,omitempty"`
	Tags                  []string    `json:"tags,omitempty"`
	Paused                bool        `json:"paused,omitempty"`
	Version               string      `json:"version,omitempty"`
}

// Checks represents a list of checks in Onfido API
type Checks struct {
	Checks []*Check `json:"checks"`
}

// CreateCheck creates a new check for the provided applicant.
// see https://documentation.onfido.com/?shell#create-check
func (c *Client) CreateCheck(ctx context.Context, applicantID string, cr CheckRequest) (*Check, error) {
	jsonStr, err := json.Marshal(cr)
	if err != nil {
		return nil, err
	}

	req, err := c.newRequest("POST", "/checks", bytes.NewBuffer(jsonStr))
	if err != nil {
		return nil, err
	}

	var resp Check
	_, err = c.do(ctx, req, &resp)
	return &resp, err
}

// GetCheck retrieves a check for the provided applicant by its ID.
// see https://documentation.onfido.com/?shell#retrieve-check
func (c *Client) GetCheck(ctx context.Context, id string) (*CheckRetrieved, error) {
	req, err := c.newRequest("GET", "/checks/"+id, nil)
	if err != nil {
		return nil, err
	}

	var resp CheckRetrieved
	_, err = c.do(ctx, req, &resp)
	return &resp, err
}

// GetCheckExpanded retrieves a check for the provided applicant by its ID, with
// the Check's Reports expanded within the returned Check object.
// see https://documentation.onfido.com/?shell#retrieve-check (Shell) but refer to the JSON
// response object for https://documentation.onfido.com/?php#check-object (PHP) for the expanded contents.
func (c *Client) GetCheckExpanded(ctx context.Context, id string) (*Check, error) {
	// Get the CheckRetrieved object. This only includes Report IDs, not the expanded Report objects.
	chkRetrieved, err := c.GetCheck(ctx, id)
	if err != nil {
		return nil, err
	}

	// Build a regular Check object, this is what will be returned assuming there is no error.
	check := Check{
		ID:                    chkRetrieved.ID,
		CreatedAt:             chkRetrieved.CreatedAt,
		Href:                  chkRetrieved.Href,
		ApplicantID:           chkRetrieved.ApplicantID,
		ApplicantProvidesData: chkRetrieved.ApplicantProvidesData,
		Status:                chkRetrieved.Status,
		Result:                chkRetrieved.Result,
		FormURI:               chkRetrieved.FormURI,
		RedirectURI:           chkRetrieved.RedirectURI,
		ResultsURI:            chkRetrieved.ResultsURI,
		Reports:               make([]*Report, len(chkRetrieved.ReportIDs)),
		Tags:                  chkRetrieved.Tags,
		Paused:                chkRetrieved.Paused,
		Version:               chkRetrieved.Version,
	}

	// For each Report ID in the CheckRetrieved object, fetch (expand) the Report
	// into the returned Check object.
	for i, reportID := range chkRetrieved.ReportIDs {
		rep, err := c.GetReport(ctx, reportID)
		if err != nil {
			return nil, err
		}
		check.Reports[i] = rep
	}
	return &check, nil
}

// ResumeCheck resumes a paused check by its ID.
// see https://documentation.onfido.com/?shell#resume-check
func (c *Client) ResumeCheck(ctx context.Context, id string) (*Check, error) {
	req, err := c.newRequest("POST", "/checks/"+id+"/resume", nil)
	if err != nil {
		return nil, err
	}

	var resp Check
	_, err = c.do(ctx, req, &resp)
	return &resp, err
}

// CheckIter represents a check iterator
type CheckIter struct {
	*iter
}

// Check returns the current item in the iterator as a Check.
func (i *CheckIter) Check() *Check {
	return i.Current().(*Check)
}

// ListChecks retrieves the list of checks for the provided applicant.
// see https://documentation.onfido.com/?shell#list-checks
func (c *Client) ListChecks(applicantID string) *CheckIter {
	handler := func(body []byte) ([]interface{}, error) {
		var r Checks
		if err := json.Unmarshal(body, &r); err != nil {
			return nil, err
		}

		values := make([]interface{}, len(r.Checks))
		for i, v := range r.Checks {
			values[i] = v
		}
		return values, nil
	}

	return &CheckIter{&iter{
		c:       c,
		nextURL: "/applicants/" + applicantID + "/checks",
		handler: handler,
	}}
}
