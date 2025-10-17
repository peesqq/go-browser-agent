package agentv2

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	bw "github.com/hang-ma/go-browser-agent/internal/browser"
	pw "github.com/playwright-community/playwright-go"
)

type WebTools struct{ B *bw.Browser }

func (t *WebTools) Navigate(ctx context.Context, args map[string]any) (string, error) {
	url, _ := args["url"].(string)
	if url == "" {
		return "", fmt.Errorf("url is required")
	}
	if err := t.B.Goto(url); err != nil {
		return "", err
	}
	t.B.WaitIdle()
	return t.Observe(ctx, map[string]any{"scope": "view"})
}

func (t *WebTools) Click(ctx context.Context, args map[string]any) (string, error) {
	page := t.B.Page()
	if page == nil {
		return "", fmt.Errorf("page is nil")
	}

	if sel, _ := args["selector"].(string); sel != "" {
		if err := page.Click(sel, pw.PageClickOptions{Timeout: pw.Float(8000)}); err != nil {
			return "", err
		}
	} else if txt, _ := args["text"].(string); txt != "" {
		loc := page.Locator(fmt.Sprintf("text=%s", txt))
		if err := loc.First().Click(); err != nil {
			return "", err
		}
	} else {
		return "", fmt.Errorf("selector or text is required")
	}

	t.B.WaitIdle()
	return t.Observe(ctx, map[string]any{"scope": "view"})
}

func (t *WebTools) Type(ctx context.Context, args map[string]any) (string, error) {
	page := t.B.Page()
	if page == nil {
		return "", fmt.Errorf("page is nil")
	}

	sel, _ := args["selector"].(string)
	txt, _ := args["text"].(string)
	submit, _ := args["submit"].(bool)

	if sel == "" {
		return "", fmt.Errorf("selector is required")
	}
	if err := page.Fill(sel, txt); err != nil {
		return "", err
	}
	if submit {
		if err := page.Press(sel, "Enter"); err != nil {
			return "", err
		}
	}

	t.B.WaitIdle()
	return t.Observe(ctx, map[string]any{"scope": "view"})
}

func (t *WebTools) Observe(ctx context.Context, args map[string]any) (string, error) {
	page := t.B.Page()
	if page == nil {
		return "", fmt.Errorf("page is nil")
	}

	scope, _ := args["scope"].(string)
	js := `(() => {
      const vis = Array.from(document.querySelectorAll('[role],button,a,input,select,textarea'))
        .slice(0, 200)
        .map(el => ({
          role: el.getAttribute('role')||el.tagName.toLowerCase(),
          name: (el.innerText||'').trim().slice(0,80) || el.getAttribute('aria-label') || el.getAttribute('name') || '',
          id: el.id || '',
          cls: el.className || '',
          sel: (el.tagName.toLowerCase() + (el.id? '#'+el.id : ''))
        }));
      const title = document.title;
      const url = location.href;
      return JSON.stringify({title,url,vis});
    })()`

	res, err := page.Evaluate(js)
	if err != nil {
		return "", err
	}

	if scope == "full" {
		_ = t.B.Screenshot(fmt.Sprintf("run_artifacts/observe_%d.png", time.Now().Unix()))
	}

	switch v := res.(type) {
	case string:
		return v, nil
	default:
		b, _ := json.Marshal(v)
		return string(b), nil
	}
}
