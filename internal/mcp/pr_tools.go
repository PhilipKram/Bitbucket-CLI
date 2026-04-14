package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/PhilipKram/bitbucket-cli/internal/api"
)

// PRListHandler handles the pr_list tool invocation.
func PRListHandler(ctx context.Context, args map[string]interface{}) ([]Content, error) {
	// Extract required parameter
	repository, ok := args["repository"].(string)
	if !ok || repository == "" {
		return nil, fmt.Errorf("repository parameter is required")
	}
	if err := validateRepoArg(repository); err != nil {
		return nil, err
	}

	// Extract optional parameters
	state, _ := args["state"].(string)
	page := 1
	if pageNum, ok := args["page"].(float64); ok {
		page = int(pageNum)
	}

	// Create API client
	client, err := GetClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create API client: %w", err)
	}

	// Build API path
	path := fmt.Sprintf("/repositories/%s/pullrequests?pagelen=25&page=%d", repository, page)
	if state != "" {
		path += "&state=" + url.QueryEscape(strings.ToUpper(state))
	}

	// Fetch pull requests
	prs, err := api.GetPaginated[PullRequest](client, path)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch pull requests: %w", err)
	}

	// Convert to JSON
	data, err := json.Marshal(prs)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal pull requests: %w", err)
	}

	return []Content{NewTextContent(string(data))}, nil
}

// PRViewHandler handles the pr_view tool invocation.
func PRViewHandler(ctx context.Context, args map[string]interface{}) ([]Content, error) {
	// Extract required parameters
	repository, ok := args["repository"].(string)
	if !ok || repository == "" {
		return nil, fmt.Errorf("repository parameter is required")
	}
	if err := validateRepoArg(repository); err != nil {
		return nil, err
	}

	prID, ok := args["pr_id"].(string)
	if !ok || prID == "" {
		return nil, fmt.Errorf("pr_id parameter is required")
	}

	// Create API client
	client, err := GetClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create API client: %w", err)
	}

	// Fetch pull request
	path := fmt.Sprintf("/repositories/%s/pullrequests/%s", repository, prID)
	data, err := client.Get(path)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch pull request: %w", err)
	}

	var pr PullRequest
	if err := json.Unmarshal(data, &pr); err != nil {
		return nil, fmt.Errorf("failed to unmarshal pull request: %w", err)
	}

	// Convert to JSON
	result, err := json.Marshal(pr)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal pull request: %w", err)
	}

	return []Content{NewTextContent(string(result))}, nil
}

// PRCreateHandler handles the pr_create tool invocation.
func PRCreateHandler(ctx context.Context, args map[string]interface{}) ([]Content, error) {
	// Extract required parameters
	repository, ok := args["repository"].(string)
	if !ok || repository == "" {
		return nil, fmt.Errorf("repository parameter is required")
	}
	if err := validateRepoArg(repository); err != nil {
		return nil, err
	}

	title, ok := args["title"].(string)
	if !ok || title == "" {
		return nil, fmt.Errorf("title parameter is required")
	}

	source, ok := args["source"].(string)
	if !ok || source == "" {
		return nil, fmt.Errorf("source parameter is required")
	}

	// Extract optional parameters
	description, _ := args["description"].(string)
	destination, _ := args["destination"].(string)
	closeBranch := false
	if cb, ok := args["close_branch"].(bool); ok {
		closeBranch = cb
	}

	// Create API client
	client, err := GetClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create API client: %w", err)
	}

	// Build request body
	body := map[string]interface{}{
		"title":               title,
		"description":         description,
		"close_source_branch": closeBranch,
		"source": map[string]interface{}{
			"branch": map[string]string{"name": source},
		},
	}
	if destination != "" {
		body["destination"] = map[string]interface{}{
			"branch": map[string]string{"name": destination},
		}
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	// Create pull request
	path := fmt.Sprintf("/repositories/%s/pullrequests", repository)
	data, err := client.Post(path, string(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create pull request: %w", err)
	}

	var pr PullRequest
	if err := json.Unmarshal(data, &pr); err != nil {
		return nil, fmt.Errorf("failed to unmarshal created pull request: %w", err)
	}

	// Convert to JSON
	result, err := json.Marshal(pr)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal pull request: %w", err)
	}

	return []Content{NewTextContent(string(result))}, nil
}

// PRApproveHandler handles the pr_approve tool invocation.
func PRApproveHandler(ctx context.Context, args map[string]interface{}) ([]Content, error) {
	// Extract required parameters
	repository, ok := args["repository"].(string)
	if !ok || repository == "" {
		return nil, fmt.Errorf("repository parameter is required")
	}
	if err := validateRepoArg(repository); err != nil {
		return nil, err
	}

	prID, ok := args["pr_id"].(string)
	if !ok || prID == "" {
		return nil, fmt.Errorf("pr_id parameter is required")
	}

	// Create API client
	client, err := GetClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create API client: %w", err)
	}

	// Approve pull request
	path := fmt.Sprintf("/repositories/%s/pullrequests/%s/approve", repository, prID)
	data, err := client.Post(path, "")
	if err != nil {
		return nil, fmt.Errorf("failed to approve pull request: %w", err)
	}

	return []Content{NewTextContent(string(data))}, nil
}

// PRMergeHandler handles the pr_merge tool invocation.
func PRMergeHandler(ctx context.Context, args map[string]interface{}) ([]Content, error) {
	// Extract required parameters
	repository, ok := args["repository"].(string)
	if !ok || repository == "" {
		return nil, fmt.Errorf("repository parameter is required")
	}
	if err := validateRepoArg(repository); err != nil {
		return nil, err
	}

	prID, ok := args["pr_id"].(string)
	if !ok || prID == "" {
		return nil, fmt.Errorf("pr_id parameter is required")
	}

	// Extract optional parameters
	mergeStrategy, _ := args["merge_strategy"].(string)
	closeBranch := false
	if cb, ok := args["close_source_branch"].(bool); ok {
		closeBranch = cb
	}

	// Create API client
	client, err := GetClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create API client: %w", err)
	}

	// Build request body
	body := map[string]interface{}{}
	if mergeStrategy != "" {
		body["merge_strategy"] = mergeStrategy
	}
	if closeBranch {
		body["close_source_branch"] = true
	}

	jsonBody := ""
	if len(body) > 0 {
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		jsonBody = string(bodyBytes)
	}

	// Merge pull request
	path := fmt.Sprintf("/repositories/%s/pullrequests/%s/merge", repository, prID)
	data, err := client.Post(path, jsonBody)
	if err != nil {
		return nil, fmt.Errorf("failed to merge pull request: %w", err)
	}

	var pr PullRequest
	if err := json.Unmarshal(data, &pr); err != nil {
		return nil, fmt.Errorf("failed to unmarshal merged pull request: %w", err)
	}

	result, err := json.Marshal(pr)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal pull request: %w", err)
	}

	return []Content{NewTextContent(string(result))}, nil
}

// PRDeclineHandler handles the pr_decline tool invocation.
func PRDeclineHandler(ctx context.Context, args map[string]interface{}) ([]Content, error) {
	// Extract required parameters
	repository, ok := args["repository"].(string)
	if !ok || repository == "" {
		return nil, fmt.Errorf("repository parameter is required")
	}
	if err := validateRepoArg(repository); err != nil {
		return nil, err
	}

	prID, ok := args["pr_id"].(string)
	if !ok || prID == "" {
		return nil, fmt.Errorf("pr_id parameter is required")
	}

	// Create API client
	client, err := GetClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create API client: %w", err)
	}

	// Decline pull request
	path := fmt.Sprintf("/repositories/%s/pullrequests/%s/decline", repository, prID)
	data, err := client.Post(path, "")
	if err != nil {
		return nil, fmt.Errorf("failed to decline pull request: %w", err)
	}

	var pr PullRequest
	if err := json.Unmarshal(data, &pr); err != nil {
		return nil, fmt.Errorf("failed to unmarshal declined pull request: %w", err)
	}

	result, err := json.Marshal(pr)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal pull request: %w", err)
	}

	return []Content{NewTextContent(string(result))}, nil
}

// PRDiffHandler handles the pr_diff tool invocation.
func PRDiffHandler(ctx context.Context, args map[string]interface{}) ([]Content, error) {
	// Extract required parameters
	repository, ok := args["repository"].(string)
	if !ok || repository == "" {
		return nil, fmt.Errorf("repository parameter is required")
	}
	if err := validateRepoArg(repository); err != nil {
		return nil, err
	}

	prID, ok := args["pr_id"].(string)
	if !ok || prID == "" {
		return nil, fmt.Errorf("pr_id parameter is required")
	}

	// Create API client
	client, err := GetClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create API client: %w", err)
	}

	// Fetch pull request diff (returns plain text)
	path := fmt.Sprintf("/repositories/%s/pullrequests/%s/diff", repository, prID)
	data, err := client.Get(path)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch pull request diff: %w", err)
	}

	return []Content{NewTextContent(string(data))}, nil
}

// PRCommentHandler handles the pr_comment tool invocation.
func PRCommentHandler(ctx context.Context, args map[string]interface{}) ([]Content, error) {
	// Extract required parameters
	repository, ok := args["repository"].(string)
	if !ok || repository == "" {
		return nil, fmt.Errorf("repository parameter is required")
	}
	if err := validateRepoArg(repository); err != nil {
		return nil, err
	}

	prID, ok := args["pr_id"].(string)
	if !ok || prID == "" {
		return nil, fmt.Errorf("pr_id parameter is required")
	}

	content, ok := args["content"].(string)
	if !ok || content == "" {
		return nil, fmt.Errorf("content parameter is required")
	}

	// Create API client
	client, err := GetClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create API client: %w", err)
	}

	// Build request body
	body := map[string]interface{}{
		"content": map[string]string{
			"raw": content,
		},
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	// Create comment
	path := fmt.Sprintf("/repositories/%s/pullrequests/%s/comments", repository, prID)
	data, err := client.Post(path, string(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create pull request comment: %w", err)
	}

	return []Content{NewTextContent(string(data))}, nil
}

// PRCommentsListHandler handles the pr_comments tool invocation.
func PRCommentsListHandler(ctx context.Context, args map[string]interface{}) ([]Content, error) {
	repository, ok := args["repository"].(string)
	if !ok || repository == "" {
		return nil, fmt.Errorf("repository parameter is required")
	}
	if err := validateRepoArg(repository); err != nil {
		return nil, err
	}

	prID, ok := args["pr_id"].(string)
	if !ok || prID == "" {
		return nil, fmt.Errorf("pr_id parameter is required")
	}

	client, err := GetClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create API client: %w", err)
	}

	path := fmt.Sprintf("/repositories/%s/pullrequests/%s/comments?pagelen=100", repository, prID)
	data, err := client.Get(path)
	if err != nil {
		return nil, fmt.Errorf("failed to list pull request comments: %w", err)
	}

	// Parse the paginated response to extract comments
	var response struct {
		Values []struct {
			ID      int `json:"id"`
			Content struct {
				Raw string `json:"raw"`
			} `json:"content"`
			User struct {
				DisplayName string `json:"display_name"`
				Nickname    string `json:"nickname"`
			} `json:"user"`
			CreatedOn string `json:"created_on"`
			UpdatedOn string `json:"updated_on"`
			Inline    *struct {
				Path string `json:"path"`
				From *int   `json:"from"`
				To   *int   `json:"to"`
			} `json:"inline"`
			Parent *struct {
				ID int `json:"id"`
			} `json:"parent"`
			Deleted bool `json:"deleted"`
		} `json:"values"`
	}
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse comments: %w", err)
	}

	// Format comments for readability
	var sb strings.Builder
	for _, c := range response.Values {
		if c.Deleted {
			continue
		}
		author := c.User.DisplayName
		if author == "" {
			author = c.User.Nickname
		}

		if c.Inline != nil {
			sb.WriteString(fmt.Sprintf("### Inline comment by %s on `%s`", author, c.Inline.Path))
			if c.Inline.To != nil {
				sb.WriteString(fmt.Sprintf(" (line %d)", *c.Inline.To))
			}
			sb.WriteString("\n")
		} else {
			sb.WriteString(fmt.Sprintf("### Comment by %s", author))
			if c.Parent != nil {
				sb.WriteString(fmt.Sprintf(" (reply to #%d)", c.Parent.ID))
			}
			sb.WriteString("\n")
		}
		sb.WriteString(fmt.Sprintf("*%s*\n\n", c.CreatedOn))
		sb.WriteString(c.Content.Raw)
		sb.WriteString("\n\n---\n\n")
	}

	if sb.Len() == 0 {
		return []Content{NewTextContent("No comments on this pull request.")}, nil
	}

	return []Content{NewTextContent(sb.String())}, nil
}

// PREditHandler handles the pr_edit tool invocation.
func PREditHandler(ctx context.Context, args map[string]interface{}) ([]Content, error) {
	repository, ok := args["repository"].(string)
	if !ok || repository == "" {
		return nil, fmt.Errorf("repository parameter is required")
	}
	if err := validateRepoArg(repository); err != nil {
		return nil, err
	}

	prID, ok := args["pr_id"].(string)
	if !ok || prID == "" {
		return nil, fmt.Errorf("pr_id parameter is required")
	}

	body := map[string]interface{}{}
	if title, ok := args["title"].(string); ok && title != "" {
		body["title"] = title
	}
	if description, ok := args["description"].(string); ok {
		body["description"] = description
	}
	if destination, ok := args["destination"].(string); ok && destination != "" {
		body["destination"] = map[string]interface{}{
			"branch": map[string]string{"name": destination},
		}
	}
	if closeBranch, ok := args["close_source_branch"].(bool); ok {
		body["close_source_branch"] = closeBranch
	}

	if len(body) == 0 {
		return nil, fmt.Errorf("no changes specified; use title, description, destination, or close_source_branch")
	}

	client, err := GetClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create API client: %w", err)
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	path := fmt.Sprintf("/repositories/%s/pullrequests/%s", repository, prID)
	data, err := client.Put(path, string(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to update pull request: %w", err)
	}

	var pr PullRequest
	if err := json.Unmarshal(data, &pr); err != nil {
		return nil, fmt.Errorf("failed to unmarshal pull request: %w", err)
	}

	result, err := json.Marshal(pr)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal pull request: %w", err)
	}

	return []Content{NewTextContent(string(result))}, nil
}

// PRUnapproveHandler handles the pr_unapprove tool invocation.
func PRUnapproveHandler(ctx context.Context, args map[string]interface{}) ([]Content, error) {
	repository, ok := args["repository"].(string)
	if !ok || repository == "" {
		return nil, fmt.Errorf("repository parameter is required")
	}
	if err := validateRepoArg(repository); err != nil {
		return nil, err
	}

	prID, ok := args["pr_id"].(string)
	if !ok || prID == "" {
		return nil, fmt.Errorf("pr_id parameter is required")
	}

	client, err := GetClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create API client: %w", err)
	}

	path := fmt.Sprintf("/repositories/%s/pullrequests/%s/approve", repository, prID)
	_, err = client.Delete(path)
	if err != nil {
		return nil, fmt.Errorf("failed to unapprove pull request: %w", err)
	}

	return []Content{NewTextContent(fmt.Sprintf("Approval removed from PR #%s", prID))}, nil
}

// PRActivityHandler handles the pr_activity tool invocation.
func PRActivityHandler(ctx context.Context, args map[string]interface{}) ([]Content, error) {
	repository, ok := args["repository"].(string)
	if !ok || repository == "" {
		return nil, fmt.Errorf("repository parameter is required")
	}
	if err := validateRepoArg(repository); err != nil {
		return nil, err
	}

	prID, ok := args["pr_id"].(string)
	if !ok || prID == "" {
		return nil, fmt.Errorf("pr_id parameter is required")
	}

	client, err := GetClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create API client: %w", err)
	}

	path := fmt.Sprintf("/repositories/%s/pullrequests/%s/activity?pagelen=50", repository, prID)
	data, err := client.Get(path)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch pull request activity: %w", err)
	}

	var paginated struct {
		Values json.RawMessage `json:"values"`
	}
	if err := json.Unmarshal(data, &paginated); err != nil {
		return nil, fmt.Errorf("failed to parse activity response: %w", err)
	}

	var activities []struct {
		Update *struct {
			State  string `json:"state"`
			Author struct {
				DisplayName string `json:"display_name"`
			} `json:"author"`
			Date string `json:"date"`
		} `json:"update"`
		Approval *struct {
			User struct {
				DisplayName string `json:"display_name"`
			} `json:"user"`
			Date string `json:"date"`
		} `json:"approval"`
		Comment *struct {
			User struct {
				DisplayName string `json:"display_name"`
			} `json:"user"`
			Content struct {
				Raw string `json:"raw"`
			} `json:"content"`
			CreatedOn string `json:"created_on"`
		} `json:"comment"`
	}
	if err := json.Unmarshal(paginated.Values, &activities); err != nil {
		return nil, fmt.Errorf("failed to parse activities: %w", err)
	}

	var sb strings.Builder
	for _, a := range activities {
		switch {
		case a.Update != nil:
			date := a.Update.Date
			if len(date) > 10 {
				date = date[:10]
			}
			sb.WriteString(fmt.Sprintf("[%s] %s changed state to %s\n", date, a.Update.Author.DisplayName, a.Update.State))
		case a.Approval != nil:
			date := a.Approval.Date
			if len(date) > 10 {
				date = date[:10]
			}
			sb.WriteString(fmt.Sprintf("[%s] %s approved\n", date, a.Approval.User.DisplayName))
		case a.Comment != nil:
			date := a.Comment.CreatedOn
			if len(date) > 10 {
				date = date[:10]
			}
			raw := a.Comment.Content.Raw
			if len(raw) > 80 {
				raw = raw[:80] + "..."
			}
			sb.WriteString(fmt.Sprintf("[%s] %s commented: %s\n", date, a.Comment.User.DisplayName, raw))
		}
	}

	if sb.Len() == 0 {
		return []Content{NewTextContent("No activity on this pull request.")}, nil
	}

	return []Content{NewTextContent(sb.String())}, nil
}

// PullRequest represents a Bitbucket pull request.
// This is copied from cmd/pr/pr.go to avoid circular dependencies.
type PullRequest struct {
	ID          int    `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	State       string `json:"state"`
	CreatedOn   string `json:"created_on"`
	UpdatedOn   string `json:"updated_on"`
	Author      struct {
		DisplayName string `json:"display_name"`
	} `json:"author"`
	Source struct {
		Branch struct {
			Name string `json:"name"`
		} `json:"branch"`
		Repository struct {
			FullName string `json:"full_name"`
		} `json:"repository"`
	} `json:"source"`
	Destination struct {
		Branch struct {
			Name string `json:"name"`
		} `json:"branch"`
	} `json:"destination"`
	CloseSourceBranch bool `json:"close_source_branch"`
	MergeCommit       *struct {
		Hash string `json:"hash"`
	} `json:"merge_commit"`
	CommentCount int `json:"comment_count"`
	TaskCount    int `json:"task_count"`
	Links        struct {
		HTML struct {
			Href string `json:"href"`
		} `json:"html"`
	} `json:"links"`
	Reviewers []struct {
		DisplayName string `json:"display_name"`
		UUID        string `json:"uuid"`
	} `json:"reviewers"`
	Participants []struct {
		User struct {
			DisplayName string `json:"display_name"`
		} `json:"user"`
		Role     string `json:"role"`
		Approved bool   `json:"approved"`
	} `json:"participants"`
}
