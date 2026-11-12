package api

import (
	"context"
	"math"
	"net/url"
	"reflect"
	"regexp"
	"strings"
	"uuid"

	"gitverse.ru/zagvozdeen/agenda/internal/models"
	"gitverse.ru/zagvozdeen/agenda/internal/models/enums"
)

var codeSHA = regexp.MustCompile("^([0-9a-f]{40}|[0-9a-f]{64})$")

type AddCodeLinkInput struct {
	TaskID    string             `json:"task_id" jsonschema:"Task to attach the commit or pull request to."`
	Type      enums.CodeLinkType `json:"type" jsonschema:"Link type: commit or pull_request."`
	URL       string             `json:"url" jsonschema:"HTTP or HTTPS URL of the commit or pull request."`
	BaseSHA   *string            `json:"base_sha,omitempty" jsonschema:"Full SHA of the before state; for a PR, use the merge base used to calculate its diff."`
	HeadSHA   *string            `json:"head_sha,omitempty" jsonschema:"Full SHA of the after state; for a commit link, the commit itself."`
	Additions *int               `json:"additions,omitempty" jsonschema:"Added lines; provide together with deletions or omit both."`
	Deletions *int               `json:"deletions,omitempty" jsonschema:"Deleted lines; provide together with additions or omit both."`
}

type CodeLinkInput struct {
	CodeLinkID uuid.UUID `json:"code_link_id" jsonschema:"Code link ID."`
}

type UpdateCodeLinkInput struct {
	CodeLinkID uuid.UUID           `json:"code_link_id" jsonschema:"Code link ID."`
	Type       *enums.CodeLinkType `json:"type,omitempty" jsonschema:"New link type."`
	URL        *string             `json:"url,omitempty" jsonschema:"New HTTP or HTTPS URL."`
	BaseSHA    Optional[string]    `json:"base_sha,omitzero" jsonschema:"Full SHA of the before state; null clears it."`
	HeadSHA    Optional[string]    `json:"head_sha,omitzero" jsonschema:"Full SHA of the after state; null clears it."`
	Additions  Optional[int]       `json:"additions,omitzero" jsonschema:"Added lines; set or clear together with deletions."`
	Deletions  Optional[int]       `json:"deletions,omitzero" jsonschema:"Deleted lines; set or clear together with additions."`
}

func (a *API) registerCodeLinks() {
	registerTool(a, "add_code_link", "Attach a commit or pull request to a task, optionally with SHA boundaries and line counts. Counts are supplied by the caller, not fetched automatically. Requires board membership.", false, false, a.AddCodeLink)
	registerTool(a, "update_code_link", "Update a code link. Changing its URL, type or SHAs clears statistics unless both counts are supplied. Requires board membership.", false, false, a.UpdateCodeLink)
	registerTool(a, "delete_code_link", "Remove a code link and its contribution to task diff totals. Requires board membership.", false, false, a.DeleteCodeLink)
}

func validCodeLink(link *models.CodeLink) error {
	if link.Type != enums.CodeLinkTypeCommit && link.Type != enums.CodeLinkTypePullRequest {
		return invalid("code link type must be commit or pull_request")
	}
	u, err := url.Parse(strings.TrimSpace(link.URL))
	if err != nil || u.Hostname() == "" || (u.Scheme != "https" && u.Scheme != "http") || u.User != nil {
		return invalid("code link must be an HTTP or HTTPS URL without credentials")
	}
	u.Host = strings.ToLower(u.Host)
	u.Path = strings.TrimRight(u.Path, "/")
	u.RawPath, u.Fragment, u.RawFragment = "", "", ""
	link.URL = u.String()
	if len(link.URL) > 2048 {
		return invalid("code link URL must not exceed 2048 bytes")
	}
	for _, sha := range []*string{link.BaseSHA, link.HeadSHA} {
		if sha != nil {
			*sha = strings.ToLower(strings.TrimSpace(*sha))
			if !codeSHA.MatchString(*sha) {
				return invalid("SHA must be a full 40 or 64 character hexadecimal hash")
			}
		}
	}
	if (link.Additions == nil) != (link.Deletions == nil) {
		return invalid("additions and deletions must both be provided or both be null")
	}
	for _, count := range []*int{link.Additions, link.Deletions} {
		if count != nil && (*count < 0 || *count > math.MaxInt32) {
			return invalid("line counts must be nonnegative 32-bit integers")
		}
	}
	return nil
}

func (a *API) AddCodeLink(ctx context.Context, input AddCodeLinkInput) (any, error) {
	link := &models.CodeLink{TaskID: input.TaskID, Type: input.Type, URL: input.URL, BaseSHA: input.BaseSHA, HeadSHA: input.HeadSHA, Additions: input.Additions, Deletions: input.Deletions, CreatedBy: actor(ctx).ID}
	if err := validCodeLink(link); err != nil {
		return nil, err
	}
	if _, err := a.writableTask(ctx, input.TaskID); err != nil {
		return nil, err
	}
	if err := a.store.CreateCodeLink(ctx, link); err != nil {
		return nil, err
	}
	if err := a.event(ctx, link.TaskID, enums.EventTypeLinked, "code_link", nil, link); err != nil {
		return nil, err
	}
	return link, nil
}

func (a *API) writableCodeLink(ctx context.Context, id uuid.UUID) (*models.CodeLink, error) {
	link, err := a.store.GetCodeLink(ctx, id)
	if err != nil {
		return nil, err
	}
	if _, err = a.writableTask(ctx, link.TaskID); err != nil {
		return nil, err
	}
	return a.store.GetCodeLink(ctx, id)
}

func (a *API) UpdateCodeLink(ctx context.Context, input UpdateCodeLinkInput) (any, error) {
	if input.Type == nil && input.URL == nil && !input.BaseSHA.Set && !input.HeadSHA.Set && !input.Additions.Set && !input.Deletions.Set {
		return nil, invalid("at least one code link field must be provided")
	}
	if input.Additions.Set != input.Deletions.Set {
		return nil, invalid("additions and deletions must be updated together")
	}
	link, err := a.writableCodeLink(ctx, input.CodeLinkID)
	if err != nil {
		return nil, err
	}
	before := *link
	if input.Type != nil {
		link.Type = *input.Type
	}
	if input.URL != nil {
		link.URL = *input.URL
	}
	if input.BaseSHA.Set {
		link.BaseSHA = input.BaseSHA.Value
	}
	if input.HeadSHA.Set {
		link.HeadSHA = input.HeadSHA.Value
	}
	if input.Additions.Set {
		link.Additions, link.Deletions = input.Additions.Value, input.Deletions.Value
	}
	if err = validCodeLink(link); err != nil {
		return nil, err
	}
	if !input.Additions.Set && (link.Type != before.Type || link.URL != before.URL || !reflect.DeepEqual(link.BaseSHA, before.BaseSHA) || !reflect.DeepEqual(link.HeadSHA, before.HeadSHA)) {
		link.Additions, link.Deletions = nil, nil
	}
	if reflect.DeepEqual(before, *link) {
		return link, nil
	}
	if err = a.store.UpdateCodeLink(ctx, link); err != nil {
		return nil, err
	}
	if err = a.event(ctx, link.TaskID, enums.EventTypeUpdated, "code_link", before, link); err != nil {
		return nil, err
	}
	return link, nil
}

func (a *API) DeleteCodeLink(ctx context.Context, input CodeLinkInput) (any, error) {
	link, err := a.writableCodeLink(ctx, input.CodeLinkID)
	if err != nil {
		return nil, err
	}
	if err = a.store.DeleteCodeLink(ctx, link.ID); err != nil {
		return nil, err
	}
	if err = a.event(ctx, link.TaskID, enums.EventTypeUnlinked, "code_link", link, nil); err != nil {
		return nil, err
	}
	return map[string]any{"code_link_id": link.ID, "removed": true}, nil
}
