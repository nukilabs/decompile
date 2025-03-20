package decompile

type PrimitiveKind uint8

const (
	None PrimitiveKind = iota
	PreTestedLoop
	PostTestedLoop
	EndlessLoop
	TwoWayConditional
)

func (k PrimitiveKind) String() string {
	switch k {
	case None:
		return "None"
	case PreTestedLoop:
		return "PreTestedLoop"
	case PostTestedLoop:
		return "PostTestedLoop"
	case EndlessLoop:
		return "EndlessLoop"
	case TwoWayConditional:
		return "TwoWayConditional"
	default:
		return "Unknown"
	}
}

type Primitive[N comparable] struct {
	Kind  PrimitiveKind
	Entry N
	Body  []N
	Exit  N
	Extra map[string]N
}
