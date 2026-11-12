package enums

type CodeLinkType string

const (
	CodeLinkTypeCommit      CodeLinkType = "commit"
	CodeLinkTypePullRequest CodeLinkType = "pull_request"
)
