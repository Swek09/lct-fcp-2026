-- +goose Up
ALTER TABLE profiles ADD COLUMN last_practice_at timestamptz;
CREATE TABLE challenges (
 id text PRIMARY KEY,
 employer_id text NOT NULL REFERENCES users(id),
 title text NOT NULL,
 body text NOT NULL,
 specialization text NOT NULL CHECK (specialization IN ('backend','frontend','analytics')),
 grade text NOT NULL CHECK (grade IN ('junior','middle','senior')),
 estimated_minutes integer NOT NULL CHECK (estimated_minutes BETWEEN 5 AND 30),
 created_at timestamptz NOT NULL DEFAULT now(),
 closes_at timestamptz NOT NULL DEFAULT now() + interval '7 days'
);
CREATE INDEX challenges_category_idx ON challenges(specialization,grade,closes_at);
CREATE TABLE challenge_submissions (
 id text PRIMARY KEY,
 challenge_id text NOT NULL REFERENCES challenges(id) ON DELETE CASCADE,
 candidate_id text NOT NULL REFERENCES users(id),
 answer text NOT NULL,
 status text NOT NULL DEFAULT 'submitted' CHECK (status IN ('submitted','accepted','needs_work')),
 feedback text NOT NULL DEFAULT '',
 submitted_at timestamptz NOT NULL DEFAULT now(),
 reviewed_at timestamptz,
 UNIQUE (challenge_id,candidate_id)
);
CREATE INDEX challenge_submissions_candidate_idx ON challenge_submissions(candidate_id,submitted_at DESC);
-- +goose Down
DROP TABLE challenge_submissions, challenges;
ALTER TABLE profiles DROP COLUMN last_practice_at;
