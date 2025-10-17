package agent

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/hang-ma/go-browser-agent/internal/browser"
	"github.com/hang-ma/go-browser-agent/internal/llm"
)

type Config struct {
	Browser   *browser.Browser
	LLM       llm.Client
	Artifacts string
}

type Core struct {
	cfg   Config
	tools *Tools
}

func New(cfg Config) *Core {
	return &Core{cfg: cfg, tools: NewTools(cfg.Browser)}
}

func (c *Core) Run(ctx context.Context, goal string, maxTime time.Duration) error {
	sys := `Ты — автономный веб-агент. У тебя есть действия:
- navigate {"url":string}
- click {"selector":string}
- type {"selector":string,"text":string,"submit":bool}
- observe {}
- finish {"report":string}

Работай итеративно: анализируй цель и текущее наблюдение страницы. 
На каждом шаге выводи РОВНО ОДНО действие в код-блоке (пример):

[action]
<действие> <JSON-аргументы>

Где <действие> одно из [navigate, click, type, observe, finish]. 
Никакого другого текста внутри блока.
Если требуются рискованные действия (удаление/оплата/отправка) — сначала спроси подтверждение в ответе вне блока.`

	// ...
	// тут продолжается остальной код функции

	// Первое наблюдение
	obs, _ := c.tools.Observe(ctx)
	ctxShort := SummarizeForLLM(obs)

	msgs := []llm.Message{
		{Role: "system", Content: sys},
		{Role: "user", Content: fmt.Sprintf("Цель: %s\nТекущее наблюдение:\n%s", goal, ctxShort)},
	}

	deadline := time.Now().Add(maxTime)
	step := 0
	for time.Now().Before(deadline) {
		step++
		out, err := c.cfg.LLM.Chat(ctx, sys, msgs)
		if err != nil {
			return err
		}

		action, payload := parseAction(out)
		if action == "" {
			// модель дала текст без блока — попросим явно дать действие
			msgs = append(msgs, llm.Message{Role: "assistant", Content: out})
			msgs = append(msgs, llm.Message{Role: "user", Content: "Пожалуйста, выдай следующий шаг строго в блоке ```action```."})
			continue
		}

		var observation string
		switch action {
		case "navigate":
			url := payload["url"]
			if url == "" {
				observation = "ERROR: empty url"
			} else if err := c.cfg.Browser.Goto(url); err != nil {
				observation = "ERROR: " + err.Error()
			} else {
				c.cfg.Browser.WaitIdle()
				observation, _ = c.tools.Observe(ctx)
			}
		case "click":
			selector := payload["selector"]
			if selector == "" {
				observation = "ERROR: empty selector"
			} else {
				if err := c.cfg.Browser.Page().Click(selector); err != nil {
					observation = "ERROR: " + err.Error()
				} else {
					c.cfg.Browser.WaitIdle()
					observation, _ = c.tools.Observe(ctx)
				}
			}
		case "type":
			selector := payload["selector"]
			text := payload["text"]
			submit := strings.EqualFold(payload["submit"], "true")
			if selector == "" {
				observation = "ERROR: empty selector"
			} else {
				if err := c.cfg.Browser.Page().Fill(selector, text); err != nil {
					observation = "ERROR: " + err.Error()
				} else {
					if submit {
						_ = c.cfg.Browser.Page().Keyboard().Press("Enter")
					}
					c.cfg.Browser.WaitIdle()
					observation, _ = c.tools.Observe(ctx)
				}
			}
		case "observe":
			observation, _ = c.tools.Observe(ctx)
		case "finish":
			fmt.Println("\n[ОТЧЁТ]\n" + payload["report"])
			return nil
		default:
			observation = "ERROR: unknown action"
		}

		c.tools.SaveArtifacts(fmt.Sprintf("step%02d_%s", step, action))

		msgs = append(msgs, llm.Message{Role: "assistant", Content: out})
		msgs = append(msgs, llm.Message{Role: "user", Content: "Наблюдение:\n" + SummarizeForLLM(observation)})
	}

	return errors.New("timeout")
}

var actionRe = regexp.MustCompile("(?s)```action\\s*(.*?)\\s*```")

func parseAction(s string) (string, map[string]string) {
	m := actionRe.FindStringSubmatch(s)
	if len(m) < 2 {
		return "", nil
	}
	line := strings.TrimSpace(m[1])
	parts := strings.SplitN(line, " ", 2)
	act := strings.ToLower(strings.TrimSpace(parts[0]))
	args := map[string]string{}
	if len(parts) == 2 {
		// naive JSON parse (tolerant)
		raw := strings.TrimSpace(parts[1])
		raw = strings.TrimPrefix(raw, "{")
		raw = strings.TrimSuffix(raw, "}")
		for _, kv := range strings.Split(raw, ",") {
			if strings.TrimSpace(kv) == "" {
				continue
			}
			kvp := strings.SplitN(kv, ":", 2)
			if len(kvp) != 2 {
				continue
			}
			k := strings.Trim(strings.TrimSpace(kvp[0]), `"'`)
			v := strings.Trim(strings.TrimSpace(kvp[1]), `"'`)
			args[k] = v
		}
	}
	return act, args
}
