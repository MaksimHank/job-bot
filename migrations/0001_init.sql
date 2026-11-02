-- 0001_init.sql
-- Хранилище всех откликов: и успешных, и неуспешных.
-- Ключ идемпотентности — (source, external_id).

CREATE TABLE IF NOT EXISTS applications (
                                            id           BIGSERIAL   PRIMARY KEY,

                                            source       TEXT        NOT NULL,              -- 'hh' | 'zarplata'
                                            external_id  TEXT        NOT NULL,              -- id вакансии на площадке
                                            url          TEXT        NOT NULL,              -- прямая ссылка на вакансию

                                            title        TEXT        NOT NULL,
                                            company      TEXT        NOT NULL DEFAULT '',
                                            letter       TEXT        NOT NULL DEFAULT '',   -- отправленное письмо (для аудита)

                                            status       TEXT        NOT NULL,              -- 'applied' | 'failed'
                                            error        TEXT        NOT NULL DEFAULT '',   -- текст ошибки, если status='failed'

                                            applied_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT applications_status_chk
    CHECK (status IN ('applied', 'failed'))
    );

-- Одна и та же вакансия не должна фигурировать дважды.
CREATE UNIQUE INDEX IF NOT EXISTS uniq_applications_source_external
    ON applications (source, external_id);

-- Быстрый подсчёт откликов за сегодня и отчёты по площадкам.
CREATE INDEX IF NOT EXISTS idx_applications_source_applied_at
    ON applications (source, applied_at DESC);

CREATE INDEX IF NOT EXISTS idx_applications_status
    ON applications (status);

-- Опционально: быстрый полнотекстовый поиск по письму (если понадобится).
-- CREATE INDEX IF NOT EXISTS idx_applications_letter_fts
--     ON applications USING GIN (to_tsvector('russian', letter));