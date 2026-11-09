package enums

type LinkType string

const (
	LinkTypeBlocks     LinkType = "blocks"
	LinkTypeRelatesTo  LinkType = "relates_to"
	LinkTypeDuplicates LinkType = "duplicates"
)
