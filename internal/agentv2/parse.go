package agentv2

import (
	"encoding/json"
	"errors"
	"regexp"
)

var actionRe = regexp.MustCompile("(?s)```action\\n(.*?)\\n```")

type Action struct {
	Kind string
	Args map[string]any
}

func ParseActionBlock(s string) (Action, error) {
	m := actionRe.FindStringSubmatch(s)
	if len(m) < 2 {
		return Action{}, errors.New("no action block")
	}
	line := m[1]
	i := 0
	for i < len(line) && line[i] != ' ' {
		i++
	}
	if i == 0 || i >= len(line) {
		return Action{}, errors.New("malformed action line")
	}
	kind := line[:i]
	raw := line[i+1:]
	var args map[string]any
	if err := json.Unmarshal([]byte(raw), &args); err != nil {
		return Action{}, err
	}
	return Action{Kind: kind, Args: args}, nil
}
