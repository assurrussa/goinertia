package core

// ProtocolVersion selects an explicit server/client contract. Existing
// applications retain ProtocolV2 unless they opt into ProtocolV3.
type ProtocolVersion uint8

const defaultDeferredGroup = "default"

const (
	ProtocolV2 ProtocolVersion = 2
	ProtocolV3 ProtocolVersion = 3
)

// WithProtocolVersion selects the protocol at startup. Unknown values retain
// the backwards-compatible v2 behavior.
func WithProtocolVersion(version ProtocolVersion) Option {
	return func(i *Inertia) { i.protocolVersion = version }
}

// ProtocolVersion returns the effective protocol contract.
func (i *Inertia) ProtocolVersion() ProtocolVersion {
	if i.protocolVersion == ProtocolV3 {
		return ProtocolV3
	}
	return ProtocolV2
}

// IsProtocolV3 reports whether this engine uses the v3 contract.
func (i *Inertia) IsProtocolV3() bool { return i.ProtocolVersion() == ProtocolV3 }

// WithPreserveBigIntegers opts v3 responses into native JavaScript BigInt
// preservation for integers outside JavaScript's safe integer range.
func WithPreserveBigIntegers(enabled bool) Option {
	return func(i *Inertia) { i.preserveBigIntegers = enabled }
}

type v3PageMetadata struct {
	rescuedProps        []string
	sharedProps         []string
	preserveBigIntegers bool
}

// Rescue marks a deferred value for reported, non-fatal resolution failure.
// Use Rescue(Defer(callback)) or Defer(Rescue(callback)). Failed values are
// omitted and their dotted paths are sent in rescuedProps. v3 is required.
func Rescue(value any) RescueProp { return RescueProp{Value: value} }

// RescueProp enables explicit recovery without changing DeferredProp's layout.
type RescueProp struct{ Value any }

// ScrollAt configures an explicit scroll array path. An empty path targets the
// whole value. Pagination metadata stays host supplied and framework neutral.
func ScrollAt(value any, cfg ScrollPropConfig, path string) ScrollAtProp {
	return ScrollAtProp{Value: value, Config: cfg, Path: path}
}

// ScrollAtProp is a v3 scroll value with a configurable wrapper path.
type ScrollAtProp struct {
	Value  any
	Config ScrollPropConfig
	Path   string
}
