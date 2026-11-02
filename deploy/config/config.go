package config

import (
	"fmt"
	"os"
	"time"

	"job-bot/internal/cases"
	"job-bot/internal/entities"

	yaml "pkg/mod/go.yaml.in/yaml/v3@v3.0.4"
)

// ---- YAML-схема (приватная, чтобы не тащить её наружу) ----

type rawConfig struct {
	Postgres struct {
		DSN string `yaml:"dsn"`
	} `yaml:"postgres"`

	HH struct {
		Token         string   `yaml:"token"`
		ResumeID      string   `yaml:"resume_id"`
		MaxPerDay     int      `yaml:"max_per_day"`
		ApplyInterval Duration `yaml:"apply_interval"`
	} `yaml:"hh"`

	Zarplata struct {
		Headless      bool     `yaml:"headless"`
		MaxPerDay     int      `yaml:"max_per_day"`
		ApplyInterval Duration `yaml:"apply_interval"`
	} `yaml:"zarplata"`

	Applicant struct {
		Name  string `yaml:"name"`
		Phone string `yaml:"phone"`
		Email string `yaml:"email"`
	} `yaml:"applicant"`

	Search []struct {
		Source string `yaml:"source"`
		Text   string `yaml:"text"`
		Area   string `yaml:"area"`
	} `yaml:"search"`
}

// Duration — обёртка для парсинга "2m", "30s", "1h" прямо из YAML.
type Duration struct{ time.Duration }

func (d *Duration) UnmarshalYAML(node *yaml.Node) error {
	var s string
	if err := node.Decode(&s); err != nil {
		return err
	}
	parsed, err := time.ParseDuration(s)
	if err != nil {
		return fmt.Errorf("invalid duration %q: %w", s, err)
	}
	d.Duration = parsed
	return nil
}

// ---- Публичная конфигурация, которую получают потребители ----

type Config struct {
	PostgresDSN    string
	HHToken        string
	HHResumeID     string
	Headless       bool
	ApplicantName  string
	ApplicantPhone string
	ApplicantEmail string

	Queries []entities.SearchQuery
	Limits  map[entities.Source]cases.Limit
}

// Load читает YAML по пути и накладывает env-overrides для секретов.
// Секреты не храним в репозитории: пустые значения в YAML + переменные окружения.
func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read %s: %w", path, err)
	}

	var raw rawConfig
	if err = yaml.Unmarshal(data, &raw); err != nil {
		return Config{}, fmt.Errorf("parse %s: %w", path, err)
	}

	// Env перекрывает YAML только для секретов и DSN.
	if v := os.Getenv("POSTGRES_DSN"); v != "" {
		raw.Postgres.DSN = v
	}
	if v := os.Getenv("HH_ACCESS_TOKEN"); v != "" {
		raw.HH.Token = v
	}
	if v := os.Getenv("HEADLESS"); v != "" {
		raw.Zarplata.Headless = v == "true" || v == "1"
	}

	if raw.Postgres.DSN == "" {
		return Config{}, fmt.Errorf("postgres.dsn is required")
	}
	if raw.HH.MaxPerDay <= 0 {
		return Config{}, fmt.Errorf("hh.max_per_day must be > 0")
	}
	if raw.HH.ApplyInterval.Duration <= 0 {
		return Config{}, fmt.Errorf("hh.apply_interval must be > 0")
	}
	if raw.Zarplata.MaxPerDay <= 0 {
		return Config{}, fmt.Errorf("zarplata.max_per_day must be > 0")
	}
	if raw.Zarplata.ApplyInterval.Duration <= 0 {
		return Config{}, fmt.Errorf("zarplata.apply_interval must be > 0")
	}

	cfg := Config{
		PostgresDSN:    raw.Postgres.DSN,
		HHToken:        raw.HH.Token,
		HHResumeID:     raw.HH.ResumeID,
		Headless:       raw.Zarplata.Headless,
		ApplicantName:  raw.Applicant.Name,
		ApplicantPhone: raw.Applicant.Phone,
		ApplicantEmail: raw.Applicant.Email,
		Limits: map[entities.Source]cases.Limit{
			entities.SourceHH: {
				MaxPerDay:     raw.HH.MaxPerDay,
				ApplyInterval: raw.HH.ApplyInterval.Duration,
			},
			entities.SourceZarplata: {
				MaxPerDay:     raw.Zarplata.MaxPerDay,
				ApplyInterval: raw.Zarplata.ApplyInterval.Duration,
			},
		},
	}

	for _, q := range raw.Search {
		if q.Source == "" || q.Text == "" {
			return Config{}, fmt.Errorf("search item needs source and text: %+v", q)
		}
		cfg.Queries = append(cfg.Queries, entities.SearchQuery{
			Source: entities.Source(q.Source),
			Text:   q.Text,
			Area:   q.Area,
		})
	}
	if len(cfg.Queries) == 0 {
		return Config{}, fmt.Errorf("no search queries configured")
	}
	return cfg, nil
}
