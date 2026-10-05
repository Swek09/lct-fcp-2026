-- +goose Up
CREATE TABLE users (
 id text PRIMARY KEY,
 email text UNIQUE NOT NULL,
 password_hash text NOT NULL,
 role text NOT NULL CHECK (role IN ('candidate','employer')),
 name text NOT NULL,
 verified boolean NOT NULL DEFAULT false,
 auth_version integer NOT NULL DEFAULT 1,
 consent_at timestamptz NOT NULL DEFAULT now(),
 created_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE profiles (
 user_id text PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
 headline text NOT NULL DEFAULT '',
 city text NOT NULL DEFAULT '',
 about text NOT NULL DEFAULT '',
 skills text[] NOT NULL DEFAULT '{}',
 experience_years integer NOT NULL DEFAULT 0 CHECK (experience_years BETWEEN 0 AND 60),
 specialization text NOT NULL DEFAULT '',
 grade text NOT NULL DEFAULT '',
 test_score integer NOT NULL DEFAULT 0 CHECK(test_score BETWEEN 0 AND 100),
 published boolean NOT NULL DEFAULT false,
 fsp_id text NOT NULL DEFAULT '',
 achievements jsonb NOT NULL DEFAULT '[]',
 grade_changed_at timestamptz,
 next_grade_change_at timestamptz,
 updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE companies (
 user_id text PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
 name text NOT NULL DEFAULT '',
 description text NOT NULL DEFAULT '',
 website text NOT NULL DEFAULT ''
);
CREATE TABLE email_tokens (
 digest text PRIMARY KEY,
 user_id text NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 kind text NOT NULL CHECK(kind IN ('verify','reset')),
 expires_at timestamptz NOT NULL
);
CREATE TABLE assessments (
 id text PRIMARY KEY,
 user_id text NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 specialization text NOT NULL CHECK(specialization IN ('backend','frontend','analytics')),
 grade text NOT NULL CHECK(grade IN ('junior','middle','senior')),
 template_version text NOT NULL,
 questions jsonb NOT NULL,
 answers jsonb,
 score integer,
 passed boolean,
 started_at timestamptz NOT NULL DEFAULT now(),
 expires_at timestamptz NOT NULL,
 submitted_at timestamptz
);
CREATE INDEX assessments_user_idx ON assessments(user_id,started_at DESC);
CREATE UNIQUE INDEX assessments_active_idx ON assessments(user_id) WHERE submitted_at IS NULL;
CREATE INDEX profiles_search_idx ON profiles(specialization,grade) WHERE published;
CREATE INDEX profiles_skills_idx ON profiles USING gin(skills);
CREATE TABLE invitations (
 id text PRIMARY KEY,
 employer_id text NOT NULL REFERENCES users(id),
 candidate_id text NOT NULL REFERENCES users(id),
 title text NOT NULL,
 message text NOT NULL,
 salary_min integer NOT NULL CHECK(salary_min > 0),
 salary_max integer NOT NULL CHECK(salary_max >= salary_min),
 status text NOT NULL DEFAULT 'sent' CHECK(status IN ('sent','viewed','accepted','declined')),
 created_at timestamptz NOT NULL DEFAULT now(),
 updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX invitations_candidate_idx ON invitations(candidate_id,created_at DESC);
CREATE INDEX invitations_employer_idx ON invitations(employer_id,created_at DESC);
CREATE TABLE mail_outbox (
 id bigserial PRIMARY KEY,
 recipient text NOT NULL,
 subject text NOT NULL,
 body text NOT NULL,
 attempts integer NOT NULL DEFAULT 0,
 created_at timestamptz NOT NULL DEFAULT now(),
 sent_at timestamptz
);
-- +goose Down
DROP TABLE mail_outbox, invitations, assessments, email_tokens, companies, profiles, users;
