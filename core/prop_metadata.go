package core

import "slices"

type propMetadataKind uint8

const (
	appendMetadata propMetadataKind = iota
	prependMetadata
	deepMetadata
	matchMetadata
)

type propMetadataLabel struct {
	kind propMetadataKind
	path string
}

// Wire labels cannot distinguish literal dots from nested paths. Track their
// emitting roots explicitly so overriding one root cannot remove another's
// metadata or attribute stale labels to an unrelated plain value.
func appendPropMetadata(c *State, page *PageDTO, root, path string, kind propMetadataKind) {
	if c.propMetadata == nil {
		c.propMetadata = make(map[propMetadataLabel][]string)
	}
	label := propMetadataLabel{kind: kind, path: path}
	c.propMetadata[label] = appendUnique(c.propMetadata[label], root)
	values := metadataValues(page, kind)
	*values = appendUnique(*values, path)
}

func clearMergeMetadata(c *State, page *PageDTO, root string) {
	delete(page.ScrollProps, root)
	for label, owners := range c.propMetadata {
		owners = slices.DeleteFunc(owners, func(owner string) bool { return owner == root })
		if len(owners) > 0 {
			c.propMetadata[label] = owners
			continue
		}
		delete(c.propMetadata, label)
		values := metadataValues(page, label.kind)
		*values = slices.DeleteFunc(*values, func(path string) bool { return path == label.path })
	}
}

func metadataValues(page *PageDTO, kind propMetadataKind) *[]string {
	switch kind {
	case appendMetadata:
		return &page.MergeProps
	case prependMetadata:
		return &page.PrependProps
	case deepMetadata:
		return &page.DeepMergeProps
	case matchMetadata:
		return &page.MatchPropsOn
	}
	panic("invalid prop metadata kind")
}
