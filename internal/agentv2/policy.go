package agentv2

import (
	"errors"
	"fmt"
	"strings"
)

type Policy struct {
	ModeCartOnly       bool
	RequireConfirmKinds map[string]bool
	BlockSelectors     []string
	Confirmer          func(prompt string) bool
}

func (p *Policy) Validate(a Action) error {
	k := strings.ToLower(a.Kind)
	if p.ModeCartOnly && k == "click" {
		if sel, _ := a.Args["selector"].(string); sel != "" && containsPay(sel, p.BlockSelectors) {
			return errors.New("blocked: payment/checkout selector")
		}
		if txt, _ := a.Args["text"].(string); txt != "" && containsPay(txt, p.BlockSelectors) {
			return errors.New("blocked: payment/checkout text")
		}
	}
	if p.RequireConfirmKinds[k] {
		if p.Confirmer != nil {
			if !p.Confirmer(fmt.Sprintf("Подтвердите выполнение опасного действия %s с аргументами %v (y/N)", k, a.Args)) {
				return errors.New("user declined")
			}
		} else {
			return errors.New("confirmation required")
		}
	}
	return nil
}

func containsPay(s string, block []string) bool {
	s = strings.ToLower(s)
	for _, b := range block {
		if strings.Contains(s, b) {
			return true
		}
	}
	return false
}
