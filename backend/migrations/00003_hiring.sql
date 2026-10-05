-- +goose Up
CREATE TABLE vacancies (
 id text PRIMARY KEY,
 employer_id text NOT NULL REFERENCES users(id),
 title text NOT NULL,
 description text NOT NULL,
 specialization text NOT NULL CHECK (specialization IN ('backend','frontend','analytics')),
 grade text NOT NULL CHECK (grade IN ('junior','middle','senior')),
 skills text[] NOT NULL DEFAULT '{}',
 work_format text NOT NULL CHECK (work_format IN ('remote','hybrid','office')),
 city text NOT NULL DEFAULT '',
 salary_min integer NOT NULL CHECK (salary_min > 0),
 salary_max integer NOT NULL CHECK (salary_max >= salary_min),
 status text NOT NULL DEFAULT 'draft' CHECK (status IN ('draft','published','closed')),
 created_at timestamptz NOT NULL DEFAULT now(),
 updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX vacancies_feed_idx ON vacancies(specialization,grade,created_at DESC) WHERE status='published';
CREATE TABLE applications (
 id text PRIMARY KEY,
 vacancy_id text NOT NULL REFERENCES vacancies(id),
 candidate_id text NOT NULL REFERENCES users(id),
 cover_letter text NOT NULL DEFAULT '',
 status text NOT NULL DEFAULT 'new' CHECK (status IN ('new','viewed','shortlisted','rejected','withdrawn')),
 consent_at timestamptz NOT NULL DEFAULT now(),
 created_at timestamptz NOT NULL DEFAULT now(),
 updated_at timestamptz NOT NULL DEFAULT now(),
 UNIQUE(vacancy_id,candidate_id)
);
CREATE INDEX applications_candidate_idx ON applications(candidate_id,created_at DESC);
CREATE TABLE shortlist (
 employer_id text NOT NULL REFERENCES users(id),
 candidate_id text NOT NULL REFERENCES users(id),
 created_at timestamptz NOT NULL DEFAULT now(),
 PRIMARY KEY(employer_id,candidate_id)
);
-- +goose Down
DROP TABLE shortlist, applications, vacancies;
