package hh

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"job-bot/internal/cases"
	"job-bot/internal/entities"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	http  *http.Client
	token string
}

func NewClient(accessToken string) *Client {
	return &Client{
		http:  &http.Client{Timeout: 20 * time.Second},
		token: accessToken,
	}
}

func (c *Client) Source() entities.Source {
	return entities.SourceHH
}

// Search — GET https://api.hh.ru/vacancies
func (c *Client) Search(ctx context.Context, q entities.SearchQuery) ([]entities.Vacancy, error) {
	u := "https://api.hh.ru/vacancies?per_page=50&only_with_vacancies=true&text=" +
		url.QueryEscape(q.Text)
	if q.Area != "" {
		u += "&area=" + url.QueryEscape(q.Area)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("HH-User-Agent", "JobBot/1.0 (mailto:you@example.com)")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("hh search status %d", resp.StatusCode)
	}

	var payload struct {
		Items []struct {
			ID           string `json:"id"`
			Name         string `json:"name"`
			AlternateURL string `json:"alternate_url"`
			Employer     struct {
				Name string `json:"name"`
			} `json:"employer"`
			Snippet struct {
				Requirement string `json:"requirement"`
			} `json:"snippet"`
		} `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}

	out := make([]entities.Vacancy, 0, len(payload.Items))
	for _, it := range payload.Items {
		out = append(out, entities.Vacancy{
			Source:       entities.SourceHH,
			ExternalID:   it.ID,
			URL:          it.AlternateURL,
			Title:        it.Name,
			Company:      it.Employer.Name,
			Requirements: strings.TrimSpace(it.Snippet.Requirement),
		})
	}
	return out, nil
}

// Apply — POST /negotiations
func (c *Client) Apply(ctx context.Context, v entities.Vacancy, letter string) error {
	body, _ := json.Marshal(map[string]string{
		"vacancy_id": v.ExternalID,
		"resume_id":  c.resumeID, // см. ниже — резюме лучше передавать в конструкторе
		"message":    letter,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://api.hh.ru/negotiations", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("HH-User-Agent", "JobBot/1.0 (mailto:you@example.com)")

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("hh apply status %d", resp.StatusCode)
	}
	return nil
}

var _ cases.VacancySource = (*Client)(nil)
