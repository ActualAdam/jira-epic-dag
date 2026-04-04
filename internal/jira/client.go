package jira

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	jira "github.com/ctreminiom/go-atlassian/jira/v2"
	"github.com/ctreminiom/go-atlassian/pkg/infra/models"
)

// Client wraps the go-atlassian Jira Cloud v2 client.
type Client struct {
	inner *jira.Client
}

// NewClient creates a Jira Cloud client using API token (Basic) auth.
// host should be like "https://yoursite.atlassian.net".
func NewClient(host, email, apiToken string) (*Client, error) {
	client, err := jira.New(nil, host)
	if err != nil {
		return nil, fmt.Errorf("creating jira client: %w", err)
	}

	client.Auth.SetBasicAuth(email, apiToken)

	return &Client{inner: client}, nil
}

// searchJQLResult matches the response shape of /rest/api/3/search/jql.
type searchJQLResult struct {
	Total  int                    `json:"total"`
	Issues []*models.IssueSchemeV2 `json:"issues"`
}

// MyProjectKeys returns the keys of all projects visible to the current user.
func (c *Client) MyProjectKeys(ctx context.Context) ([]string, error) {
	var keys []string
	startAt := 0
	maxResults := 50

	for {
		result, resp, err := c.inner.Project.Search(ctx, &models.ProjectSearchOptionsScheme{}, startAt, maxResults)
		if err != nil {
			body := ""
			if resp != nil {
				body = resp.Bytes.String()
			}
			return nil, fmt.Errorf("fetching projects: %w\n%s", err, body)
		}

		for _, p := range result.Values {
			keys = append(keys, p.Key)
		}

		if result.IsLast || startAt+len(result.Values) >= result.Total {
			break
		}
		startAt += len(result.Values)
	}

	return keys, nil
}

// SearchOpenEpics returns all unresolved epics from the given projects using the /rest/api/3/search/jql endpoint.
func (c *Client) SearchOpenEpics(ctx context.Context, projectKeys []string) ([]*models.IssueSchemeV2, error) {
	projectClause := ""
	if len(projectKeys) > 0 {
		projectClause = fmt.Sprintf("project IN (%s) AND ", strings.Join(projectKeys, ", "))
	}
	jql := fmt.Sprintf("%sissuetype = Epic AND resolution = Unresolved ORDER BY created DESC", projectClause)
	fields := "summary,status,assignee,project"

	var all []*models.IssueSchemeV2
	startAt := 0
	maxResults := 50

	for {
		params := url.Values{}
		params.Set("jql", jql)
		params.Set("fields", fields)
		params.Set("startAt", strconv.Itoa(startAt))
		params.Set("maxResults", strconv.Itoa(maxResults))

		endpoint := fmt.Sprintf("/rest/api/3/search/jql?%s", params.Encode())

		req, err := c.inner.NewRequest(ctx, "GET", endpoint, "", nil)
		if err != nil {
			return nil, fmt.Errorf("building search request: %w", err)
		}

		var result searchJQLResult
		resp, err := c.inner.Call(req, &result)
		if err != nil {
			body := ""
			if resp != nil {
				body = resp.Bytes.String()
			}
			return nil, fmt.Errorf("searching open epics: %w\n%s", err, body)
		}

		all = append(all, result.Issues...)

		if startAt+len(result.Issues) >= result.Total {
			break
		}
		startAt += len(result.Issues)
	}

	return all, nil
}

// SearchEpicChildren returns all child issues of the given epic key.
func (c *Client) SearchEpicChildren(ctx context.Context, epicKey string) ([]*models.IssueSchemeV2, error) {
	jql := fmt.Sprintf("parentEpic = %s ORDER BY status ASC, created ASC", epicKey)
	fields := "summary,status,assignee,issuetype"

	var all []*models.IssueSchemeV2
	startAt := 0
	maxResults := 50

	for {
		params := url.Values{}
		params.Set("jql", jql)
		params.Set("fields", fields)
		params.Set("startAt", strconv.Itoa(startAt))
		params.Set("maxResults", strconv.Itoa(maxResults))

		endpoint := fmt.Sprintf("/rest/api/3/search/jql?%s", params.Encode())

		req, err := c.inner.NewRequest(ctx, "GET", endpoint, "", nil)
		if err != nil {
			return nil, fmt.Errorf("building search request: %w", err)
		}

		var result searchJQLResult
		resp, err := c.inner.Call(req, &result)
		if err != nil {
			body := ""
			if resp != nil {
				body = resp.Bytes.String()
			}
			return nil, fmt.Errorf("searching epic children: %w\n%s", err, body)
		}

		all = append(all, result.Issues...)

		if startAt+len(result.Issues) >= result.Total {
			break
		}
		startAt += len(result.Issues)
	}

	return all, nil
}
