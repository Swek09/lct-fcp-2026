package app

import (
	"context"
	"encoding/json"
	"github.com/Swek09/lct-fcp-2026/backend/internal/matching"
	"github.com/danielgtaylor/huma/v2"
	"sort"
	"strings"
	"time"
)

type Candidate struct {
	Profile
	MatchScore int      `json:"match_score"`
	Reasons    []string `json:"reasons"`
	Email      string   `json:"email,omitempty"`
}
type SearchResponse struct {
	Candidates     []Candidate `json:"candidates"`
	Total          int         `json:"total"`
	Specialization string      `json:"specialization"`
	Grade          string      `json:"grade"`
	Skills         []string    `json:"skills"`
}

func (a *App) matchingRoutes(api huma.API) {
	type Search struct {
		Need           string `query:"need" maxLength:"2000"`
		Specialization string `query:"specialization"`
		Grade          string `query:"grade"`
		Skills         string `query:"skills" maxLength:"500"`
		FSP            bool   `query:"fsp"`
		Page           int    `query:"page" default:"1" minimum:"1" maximum:"1000"`
	}
	register(api, "GET", "/api/candidates", "Подбор по подтверждённой категории", func(ctx context.Context, in *Search) (*Response[SearchResponse], error) {
		u, err := a.require(ctx, "employer")
		if err != nil {
			return nil, err
		}
		spec, grade, skills := matching.ParseNeed(in.Need)
		if in.Specialization != "" {
			spec = in.Specialization
		}
		if in.Grade != "" {
			grade = in.Grade
		}
		if in.Skills != "" {
			skills = strings.Split(in.Skills, ",")
		}
		rows, err := a.DB.Query(ctx, `SELECT row_to_json(t) FROM (
    SELECT p.*,u.name,CASE WHEN EXISTS(SELECT 1 FROM invitations i WHERE i.candidate_id=p.user_id AND i.employer_id=$4 AND i.status='accepted') THEN u.email ELSE '' END AS email
    FROM profiles p JOIN users u ON u.id=p.user_id
    WHERE p.published AND p.grade<>'' AND u.verified
    AND ($1='' OR p.specialization=$1) AND ($2='' OR p.grade=$2)
    AND (NOT $3 OR jsonb_array_length(p.achievements)>0)
  ) t`, spec, grade, in.FSP, u.ID)
		if err != nil {
			return nil, internal(err)
		}
		defer rows.Close()
		result := []Candidate{}
		for rows.Next() {
			var raw []byte
			if err = rows.Scan(&raw); err != nil {
				return nil, internal(err)
			}
			var c Candidate
			if err = json.Unmarshal(raw, &c); err != nil {
				return nil, internal(err)
			}
			// An explicit stack filter is strict; skills parsed from a free-form
			// need only affect ranking, preserving potentially relevant candidates.
			if in.Skills != "" {
				have := map[string]bool{}
				for _, skill := range c.Skills {
					have[matching.Normalize(skill)] = true
				}
				matches := true
				for _, skill := range skills {
					if strings.TrimSpace(skill) != "" && !have[matching.Normalize(skill)] {
						matches = false
					}
				}
				if !matches {
					continue
				}
			}
			rank := matching.Rank(matching.Evidence{TestScore: c.TestScore, Skills: c.Skills, Achievements: len(c.Achievements)}, skills)
			c.MatchScore = rank.Total
			c.Reasons = rank.Reasons
			if c.LastPracticeAt != nil && time.Since(*c.LastPracticeAt) <= 30*24*time.Hour {
				c.Reasons = append(c.Reasons, "Работодатель подтвердил решение короткой задачи за последние 30 дней")
			}
			result = append(result, c)
		}
		if err = rows.Err(); err != nil {
			return nil, internal(err)
		}
		sort.SliceStable(result, func(i, j int) bool {
			if result[i].MatchScore == result[j].MatchScore {
				if result[i].LastPracticeAt != nil && time.Since(*result[i].LastPracticeAt) <= 30*24*time.Hour && (result[j].LastPracticeAt == nil || result[i].LastPracticeAt.After(*result[j].LastPracticeAt)) {
					return true
				}
				if result[j].LastPracticeAt != nil && time.Since(*result[j].LastPracticeAt) <= 30*24*time.Hour && (result[i].LastPracticeAt == nil || result[j].LastPracticeAt.After(*result[i].LastPracticeAt)) {
					return false
				}
				return result[i].UserID < result[j].UserID
			}
			return result[i].MatchScore > result[j].MatchScore
		})
		total := len(result)
		start := min((in.Page-1)*20, total)
		end := min(start+20, total)
		return ok(SearchResponse{result[start:end], total, spec, grade, skills}), nil
	})
	type Detail struct {
		ID string `path:"id"`
	}
	register(api, "GET", "/api/candidates/{id}", "Профиль с проверкой доступа к контактам", func(ctx context.Context, in *Detail) (*Response[Candidate], error) {
		u, err := a.require(ctx, "employer")
		if err != nil {
			return nil, err
		}
		c, err := readJSON[Candidate](ctx, a.DB, `SELECT row_to_json(t) FROM (
   SELECT p.*,u.name,CASE WHEN EXISTS(SELECT 1 FROM invitations i WHERE i.candidate_id=p.user_id AND i.employer_id=$2 AND i.status='accepted') THEN u.email ELSE '' END AS email
   FROM profiles p JOIN users u ON u.id=p.user_id WHERE p.user_id=$1 AND u.verified AND
   ((p.published AND p.grade<>'') OR EXISTS(SELECT 1 FROM invitations i WHERE i.candidate_id=p.user_id AND i.employer_id=$2 AND i.status='accepted'))
  ) t`, in.ID, u.ID)
		if isMissing(err) {
			return nil, huma.Error404NotFound("Профиль не найден")
		}
		if err != nil {
			return nil, internal(err)
		}
		rank := matching.Rank(matching.Evidence{TestScore: c.TestScore, Skills: c.Skills, Achievements: len(c.Achievements)}, nil)
		c.MatchScore = rank.Total
		c.Reasons = rank.Reasons
		return ok(c), nil
	})
}
