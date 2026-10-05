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
	MatchScore int            `json:"match_score"`
	Reasons    []string       `json:"reasons"`
	Email      string         `json:"email,omitempty"`
	Match      matching.Score `json:"match"`
}
type SearchResponse struct {
	Candidates     []Candidate `json:"candidates"`
	Total          int         `json:"total"`
	Specialization string      `json:"specialization"`
	Grade          string      `json:"grade"`
	Skills         []string    `json:"skills"`
	Warnings       []string    `json:"warnings"`
}

func (a *App) matchingRoutes(api huma.API) {
	type Search struct {
		Need           string `query:"need" maxLength:"2000"`
		Specialization string `query:"specialization"`
		Grade          string `query:"grade"`
		Skills         string `query:"skills" maxLength:"500"`
		FSP            bool   `query:"fsp"`
		Saved          bool   `query:"saved"`
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
		grant := strings.ReplaceAll(contactGrant, "$2", "$4")
		rows, err := a.DB.Query(ctx, `SELECT row_to_json(t) FROM (
    SELECT p.*,u.name,CASE WHEN (`+grant+`) THEN u.email ELSE '' END AS email
    FROM profiles p JOIN users u ON u.id=p.user_id
    WHERE ((p.published AND p.grade<>'' AND u.verified) OR ($5 AND (`+grant+`)))
    AND ($1='' OR p.specialization=$1) AND ($2='' OR p.grade=$2)
    AND (NOT $3 OR jsonb_array_length(p.achievements)>0)
    AND (NOT $5 OR EXISTS(SELECT 1 FROM shortlist s WHERE s.employer_id=$4 AND s.candidate_id=p.user_id))
  ) t`, spec, grade, in.FSP, u.ID, in.Saved)
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
			c.Match = rank
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
		warnings := []string{}
		if in.Need != "" && spec == "" {
			warnings = append(warnings, "Направление не определено однозначно. Выберите его в фильтре.")
		}
		if in.Need != "" && grade == "" {
			warnings = append(warnings, "Уровень не указан: показываем все подтверждённые грейды.")
		}
		return ok(SearchResponse{Candidates: result[start:end], Total: total, Specialization: spec, Grade: grade, Skills: skills, Warnings: warnings}), nil
	})
	type Detail struct {
		ID     string `path:"id"`
		Need   string `query:"need" maxLength:"2000"`
		Skills string `query:"skills" maxLength:"500"`
	}
	register(api, "GET", "/api/candidates/{id}", "Профиль с проверкой доступа к контактам", func(ctx context.Context, in *Detail) (*Response[Candidate], error) {
		u, err := a.require(ctx, "employer")
		if err != nil {
			return nil, err
		}
		c, err := readJSON[Candidate](ctx, a.DB, `SELECT row_to_json(t) FROM (
   SELECT p.*,u.name,CASE WHEN (`+contactGrant+`) THEN u.email ELSE '' END AS email
   FROM profiles p JOIN users u ON u.id=p.user_id WHERE p.user_id=$1 AND (`+candidateAccess+`)
  ) t`, in.ID, u.ID)
		if isMissing(err) {
			return nil, huma.Error404NotFound("Профиль не найден")
		}
		if err != nil {
			return nil, internal(err)
		}
		_, _, wanted := matching.ParseNeed(in.Need)
		if in.Skills != "" {
			wanted = strings.Split(in.Skills, ",")
		}
		rank := matching.Rank(matching.Evidence{TestScore: c.TestScore, Skills: c.Skills, Achievements: len(c.Achievements)}, wanted)
		c.MatchScore = rank.Total
		c.Reasons = rank.Reasons
		c.Match = rank
		return ok(c), nil
	})
}
