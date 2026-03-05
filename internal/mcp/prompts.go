package mcp

import "fmt"

// RegisterDefaultPrompts registers the built-in prompt templates with the server.
func RegisterDefaultPrompts(s *Server) {
	// review_pr - Instructions for reviewing a pull request
	s.AddPrompt(
		Prompt{
			Name:        "review_pr",
			Description: "Instructions for reviewing a Bitbucket pull request",
			Arguments: []PromptArgument{
				{Name: "repository", Description: "Repository in format workspace/repo-slug", Required: true},
				{Name: "pr_id", Description: "Pull request ID", Required: true},
			},
		},
		func(args map[string]string) (*PromptGetResult, error) {
			repo := args["repository"]
			prID := args["pr_id"]
			if repo == "" || prID == "" {
				return nil, fmt.Errorf("repository and pr_id are required")
			}

			text := fmt.Sprintf(`Please review pull request #%s in repository %s. Follow these guidelines:

1. **Code Quality**: Check for clean, readable, and maintainable code. Look for code smells, duplicated logic, and overly complex functions.

2. **Tests**: Verify that new or changed code has adequate test coverage. Check that existing tests still pass and are not broken by the changes.

3. **Security**: Look for potential security vulnerabilities such as injection flaws, hardcoded credentials, improper input validation, or insecure API usage.

4. **Performance**: Identify any potential performance issues such as N+1 queries, unnecessary allocations, or inefficient algorithms.

5. **Documentation**: Ensure public APIs and complex logic are properly documented. Check that README or docs are updated if needed.

6. **Compatibility**: Check for breaking changes to public APIs or interfaces. Verify backward compatibility where required.

Use the pr_view and pr_diff tools to examine the pull request details and changes.`, prID, repo)

			return &PromptGetResult{
				Description: fmt.Sprintf("Review guidance for PR #%s in %s", prID, repo),
				Messages: []PromptMessage{
					{
						Role:    "user",
						Content: NewTextContent(text),
					},
				},
			}, nil
		},
	)

	// explain_pipeline_failure - Instructions for diagnosing pipeline failures
	s.AddPrompt(
		Prompt{
			Name:        "explain_pipeline_failure",
			Description: "Instructions for diagnosing a Bitbucket pipeline failure",
			Arguments: []PromptArgument{
				{Name: "repository", Description: "Repository in format workspace/repo-slug", Required: true},
				{Name: "pipeline_uuid", Description: "Pipeline UUID to diagnose", Required: true},
				{Name: "step_uuid", Description: "Optional specific step UUID to focus on", Required: false},
			},
		},
		func(args map[string]string) (*PromptGetResult, error) {
			repo := args["repository"]
			pipelineUUID := args["pipeline_uuid"]
			if repo == "" || pipelineUUID == "" {
				return nil, fmt.Errorf("repository and pipeline_uuid are required")
			}

			stepContext := ""
			if stepUUID := args["step_uuid"]; stepUUID != "" {
				stepContext = fmt.Sprintf("\nFocus specifically on step %s for the root cause.", stepUUID)
			}

			text := fmt.Sprintf(`Please diagnose the pipeline failure for pipeline %s in repository %s.%s

Follow these diagnostic steps:

1. **Identify the Failed Step**: Determine which step in the pipeline failed and at what point.

2. **Analyze Error Messages**: Look at the error output and logs to understand what went wrong.

3. **Common Causes**: Consider these common failure categories:
   - Build failures (compilation errors, missing dependencies)
   - Test failures (failing assertions, flaky tests, timeout issues)
   - Deployment failures (permission issues, configuration errors, resource limits)
   - Infrastructure issues (network problems, service unavailability)

4. **Environment Context**: Check if the failure is environment-specific (e.g., works locally but fails in CI).

5. **Recent Changes**: Correlate the failure with recent commits or configuration changes.

6. **Suggested Fix**: Provide a clear recommendation for resolving the issue.

Use the pipeline_view tool to get pipeline details and step information.`, pipelineUUID, repo, stepContext)

			return &PromptGetResult{
				Description: fmt.Sprintf("Diagnostic guidance for pipeline %s in %s", pipelineUUID, repo),
				Messages: []PromptMessage{
					{
						Role:    "user",
						Content: NewTextContent(text),
					},
				},
			}, nil
		},
	)

	// summarize_issues - Instructions for summarizing repository issues
	s.AddPrompt(
		Prompt{
			Name:        "summarize_issues",
			Description: "Instructions for summarizing issues in a Bitbucket repository",
			Arguments: []PromptArgument{
				{Name: "repository", Description: "Repository in format workspace/repo-slug", Required: true},
				{Name: "state", Description: "Issue state filter (default: open)", Required: false},
			},
		},
		func(args map[string]string) (*PromptGetResult, error) {
			repo := args["repository"]
			if repo == "" {
				return nil, fmt.Errorf("repository is required")
			}

			state := args["state"]
			if state == "" {
				state = "open"
			}

			text := fmt.Sprintf(`Please summarize the %s issues in repository %s. Provide the following:

1. **Overview**: Total count of issues and a brief high-level summary of the current state.

2. **Categorization**: Group issues by type (bugs, enhancements, proposals, tasks) and priority (critical, major, minor, trivial).

3. **Key Themes**: Identify recurring themes or patterns across issues. Are there common areas of the codebase or features that have multiple issues?

4. **Critical Items**: Highlight any critical or blocking issues that need immediate attention.

5. **Staleness**: Note any issues that have been open for a long time without activity.

6. **Recommendations**: Suggest priorities for which issues to address first based on severity, impact, and effort.

Use the issue_list tool to fetch the issues.`, state, repo)

			return &PromptGetResult{
				Description: fmt.Sprintf("Summarization guidance for %s issues in %s", state, repo),
				Messages: []PromptMessage{
					{
						Role:    "user",
						Content: NewTextContent(text),
					},
				},
			}, nil
		},
	)

	// draft_pr_description - Instructions for drafting a PR description
	s.AddPrompt(
		Prompt{
			Name:        "draft_pr_description",
			Description: "Instructions for drafting a pull request description",
			Arguments: []PromptArgument{
				{Name: "repository", Description: "Repository in format workspace/repo-slug", Required: true},
				{Name: "source_branch", Description: "Source branch name", Required: false},
				{Name: "destination_branch", Description: "Destination branch name", Required: false},
			},
		},
		func(args map[string]string) (*PromptGetResult, error) {
			repo := args["repository"]
			if repo == "" {
				return nil, fmt.Errorf("repository is required")
			}

			branchContext := ""
			if source := args["source_branch"]; source != "" {
				branchContext += fmt.Sprintf("\nSource branch: %s", source)
			}
			if dest := args["destination_branch"]; dest != "" {
				branchContext += fmt.Sprintf("\nDestination branch: %s", dest)
			}

			text := fmt.Sprintf(`Please draft a pull request description for repository %s.%s

The description should follow this structure:

1. **Title**: A concise, descriptive title that summarizes the change (under 72 characters).

2. **Summary**: A brief paragraph explaining what was changed and why. Focus on the motivation and context rather than implementation details.

3. **Changes**: A bullet-point list of the key changes made:
   - What was added, modified, or removed
   - Any new dependencies introduced
   - Configuration changes

4. **Testing**: Describe how the changes were tested:
   - Unit tests added or modified
   - Manual testing performed
   - Edge cases considered

5. **Screenshots**: Note if any UI changes require screenshots (placeholder if applicable).

6. **Related Issues**: Reference any related Bitbucket issues (e.g., "Fixes #123", "Related to #456").

7. **Checklist**:
   - [ ] Code follows project conventions
   - [ ] Tests pass locally
   - [ ] Documentation updated if needed
   - [ ] No breaking changes (or documented if any)

Use the branch_list and repo_view tools to gather context about the repository and branches.`, repo, branchContext)

			return &PromptGetResult{
				Description: fmt.Sprintf("PR description drafting guidance for %s", repo),
				Messages: []PromptMessage{
					{
						Role:    "user",
						Content: NewTextContent(text),
					},
				},
			}, nil
		},
	)
}
