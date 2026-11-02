package zarplata

import (
	"job-bot/internal/cases"
	"job-bot/internal/entities"

	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	chromed "github.com/chromedp/chromedp"
)

type Browser struct {
	headless bool
}

func NewBrowser(headless bool) *Browser {
	return &Browser{
		headless: headless,
	}
}

func (b *Browser) Source() entities.Source {
	return entities.SourceZarplata
}

func (b *Browser) Search(ctx context.Context, q entities.SearchQuery) ([]entities.Vacancy, error) {
	ctx, cancel := b.spawn(ctx, 90*time.Second)
	defer cancel()

	searchURL := "https://hr.zarplata.ru/vacancies?text=" + url.QueryEscape(q.Text)
	var raw []map[string]string
	err := chromed.Run(ctx,
		chromed.Navigate(searchURL),
		chromed.WaitVisible(`[data-qa="vacancy-serp__vacancy"]`, chromed.ByQuery),
		chromed.Evaluate(`
            Array.from(document.querySelectorAll('[data-qa="vacancy-serp__vacancy"]'))
              .map(c => {
                const a = c.querySelector('a[data-qa="serp-item__title-link"]');
                const comp = c.querySelector('[data-qa="vacancy-serp__vacancy-employer"]');
                return a ? {
                  url:   a.href,
                  title: a.textContent.trim(),
                  company: comp ? comp.textContent.trim() : ''
                } : null;
              }).filter(Boolean)
        `, &raw),
	)
	if err != nil {
		return nil, fmt.Errorf("zarplata search: %w", err)
	}

	out := make([]entities.Vacancy, 0, len(raw))
	for _, r := range raw {
		id := extractID(r["url"])
		if id == "" {
			continue
		}
		out = append(out, entities.Vacancy{
			Source:     entities.SourceZarplata,
			ExternalID: id,
			URL:        r["url"],
			Title:      r["title"],
			Company:    r["company"],
		})
	}
	return out, nil
}

func (b *Browser) Apply(ctx context.Context, v entities.Vacancy, letter string) error {
	ctx, cancel := b.spawn(ctx, 60*time.Second)
	defer cancel()

	return chromed.Run(ctx,
		chromed.Navigate(v.URL),
		chromed.WaitVisible(`a[data-qa="vacancy-response-link-top"]`, chromed.ByQuery),
		chromed.Click(`a[data-qa="vacancy-response-link-top"]`, chromed.ByQuery),
		chromed.WaitVisible(`textarea[name="letter"]`, chromed.ByQuery),
		chromed.SendKeys(`textarea[name="letter"]`, letter, chromed.ByQuery),
		chromed.Click(`button[data-qa="vacancy-response-submit-popup"]`, chromed.ByQuery),
		chromed.WaitVisible(`div[data-qa="vacancy-response-success"]`, chromed.ByQuery),
	)
}

func (b *Browser) spawn(parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	opts := append(chromed.DefaultExecAllocatorOptions[:],
		chromed.Flag("headless", b.headless),
	)
	allocCtx, allocCancel := chromed.NewExecAllocator(parent, opts...)
	ctx, ctxCancel := chromed.NewContext(allocCtx)
	ctx, toCancel := context.WithTimeout(ctx, timeout)
	return ctx, func() { toCancel(); ctxCancel(); allocCancel() }
}

// extractID — вытаскивает числовой ID из URL /vacancy/1234567.
func extractID(raw string) string {
	// Простейший парсер без regexp: /vacancy/<digits>
	const marker = "/vacancy/"
	i := stringsIndex(raw, marker)
	if i < 0 {
		return ""
	}
	rest := raw[i+len(marker):]
	end := 0
	for end < len(rest) && rest[end] >= '0' && rest[end] <= '9' {
		end++
	}
	return rest[:end]
}

// Вспомогательный алиас, чтобы не тянуть strings только ради одного Index.
func stringsIndex(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

var _ = json.Marshal // на случай если понадобится
var _ cases.VacancySource = (*Browser)(nil)
