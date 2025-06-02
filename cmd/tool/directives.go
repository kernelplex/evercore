package main

type DirectiveType string

const (
	DirectiveAggregate  DirectiveType = "aggregate"
	DirectiveEvent      DirectiveType = "event"
	DirectiveStateEvent DirectiveType = "state-event"
)

type Directive struct {
	Type        DirectiveType
	Struct      string
	FilePath    string
	Package     string
	PackageName string
	Line        int
}

type LocatedDirectives struct {
	Aggregates  map[string]Directive `json:"aggregates"`
	StateEvents map[string]Directive `json:"state_events"`
	Events      map[string]Directive `json:"events"`
}

func NewLocatedDirectives() LocatedDirectives {
	return LocatedDirectives{
		Aggregates:  make(map[string]Directive),
		StateEvents: make(map[string]Directive),
		Events:      make(map[string]Directive),
	}
}

func validateDirective(directive string, filePath string, lineNum int) error {
	switch directive {
	case string(DirectiveAggregate), string(DirectiveEvent), string(DirectiveStateEvent):
		return nil
	default:
		return &ErrInvalidDirective{
			Directive: directive,
			FilePath:  filePath,
			Line:      lineNum,
		}
	}
}
