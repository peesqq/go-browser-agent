package agentv2

import (
	"context"
	"errors"
	"strings"
)

type ToolFunc func(ctx context.Context, args map[string]any) (string, error)

type Executor struct {
	Tools  map[string]ToolFunc
	Policy *Policy
}

func (e *Executor) Do(ctx context.Context, a Action) (string, error) {
	if e.Policy != nil {
		if err := e.Policy.Validate(a); err != nil {
			return "", err
		}
	}
	t, ok := e.Tools[strings.ToLower(a.Kind)]
	if !ok {
		return "", errors.New("unknown action: " + a.Kind)
	}
	return t(ctx, a.Args)
}
