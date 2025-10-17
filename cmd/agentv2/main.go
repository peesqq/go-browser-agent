package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/hang-ma/go-browser-agent/internal/agentv2"
	"github.com/hang-ma/go-browser-agent/internal/browser"
	"github.com/hang-ma/go-browser-agent/internal/llm"
)

func main() {
	var model string
	flag.StringVar(&model, "model", "z-ai/glm-4.5-air:free", "Model ID (OpenRouter/OpenAI-compatible)")
	flag.Parse()

	ctx := context.Background()

	// init LLM — ключ и BASE берутся внутри llm.NewOpenAI (openrouter headers/BASE уже прописаны)
	client, err := llm.NewOpenAI(model)
	if err != nil {
		panic(err)
	}

	// init browser
	b, err := browser.NewPlaywright(ctx, browser.Config{
		UserDataDir: "user_data",
		Headless:    false,
		SlowMo:      100,
	})
	if err != nil {
		panic(err)
	}
	defer b.Close()

	fmt.Println("✅ Инициализация завершена успешно")

	tools := &agentv2.WebTools{B: b}
	exec := &agentv2.Executor{
		Tools: map[string]agentv2.ToolFunc{
			"navigate": tools.Navigate,
			"click":    tools.Click,
			"type":     tools.Type,
			"observe":  tools.Observe,
			"finish": func(ctx context.Context, args map[string]any) (string, error) {
				report, _ := args["report"].(string)
				fmt.Println("\n=== FINISH ===\n" + report)
				return "DONE", nil
			},
		},
		Policy: &agentv2.Policy{
			ModeCartOnly: true,
			RequireConfirmKinds: map[string]bool{
				"delete": true,
				"submit": true,
			},
			BlockSelectors: []string{"pay", "checkout", "оплат", "buy", "place order", "complete order"},
			Confirmer: func(prompt string) bool {
				fmt.Printf("%s\n> ", prompt)
				in := bufio.NewReader(os.Stdin)
				text, _ := in.ReadString('\n')
				return strings.HasPrefix(strings.ToLower(strings.TrimSpace(text)), "y")
			},
		},
	}

	// === REPL: пользователь -> LLM -> действие ===
	fmt.Println("Agent v2 ready. Type your goal (Ctrl+C to exit).")
	in := bufio.NewScanner(os.Stdin)

	for {
		fmt.Print("\nYou> ")
		if !in.Scan() {
			break
		}
		goal := strings.TrimSpace(in.Text())
		if goal == "" {
			continue
		}

		sys := `Ты — автономный веб-агент. Планируй и выполняй шаги. 
Запрещено оплачивать или оформлять заказы (checkout/pay).
Отвечай только действием в markdown-код-блоке строго в формате:
action <действие> {"аргументы":значения}
Без дополнительных тегов (<action>, <kind> и т.д.).
Доступные действия: navigate, click, type, observe, finish.
Перед опасными действиями используй finish с запросом подтверждения.`

		user := "Задача: " + goal + "\nНачни с navigate/observe по необходимости."

		resp, err := client.Chat(ctx, sys, []llm.Message{
			{Role: "user", Content: user},
		})
		if err != nil {
			fmt.Println("LLM error:", err)
			continue
		}

		fmt.Println("LLM response:\n", resp)

		act, err := agentv2.ParseActionBlock(resp)
		if err != nil {
			fmt.Println("Parse error:", err)
			continue
		}

		out, err := exec.Do(ctx, act)
		if err != nil {
			fmt.Println("Ошибка выполнения:", err)
			continue
		}

		fmt.Println(out)
	}
}
