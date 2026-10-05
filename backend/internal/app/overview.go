package app

import (
	"context"
	"github.com/danielgtaylor/huma/v2"
)

type CategoryCount struct {
	Specialization string `json:"specialization"`
	Grade          string `json:"grade"`
	Count          int    `json:"count"`
}
type Overview struct {
	Invitations        map[string]int  `json:"invitations"`
	Applications       map[string]int  `json:"applications"`
	Categories         []CategoryCount `json:"categories"`
	CandidateCount     int             `json:"candidate_count"`
	PublishedVacancies int             `json:"published_vacancies"`
	SavedCandidates    int             `json:"saved_candidates"`
}

func (a *App) overviewRoutes(api huma.API) {
	register(api, "GET", "/api/overview", "Сводка собственного рабочего пространства", func(ctx context.Context, in *struct{}) (*Response[Overview], error) {
		u, err := a.require(ctx, "")
		if err != nil {
			return nil, err
		}
		out := Overview{Invitations: map[string]int{}, Applications: map[string]int{}, Categories: []CategoryCount{}}
		rows, err := a.DB.Query(ctx, "SELECT status,count(*) FROM invitations WHERE employer_id=$1 OR candidate_id=$1 GROUP BY status", u.ID)
		if err != nil {
			return nil, internal(err)
		}
		for rows.Next() {
			var s string
			var n int
			if err = rows.Scan(&s, &n); err != nil {
				rows.Close()
				return nil, internal(err)
			}
			out.Invitations[s] = n
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return nil, internal(err)
		}
		rows, err = a.DB.Query(ctx, `SELECT x.status,count(*) FROM applications x JOIN vacancies v ON v.id=x.vacancy_id WHERE ($2='employer' AND v.employer_id=$1) OR ($2='candidate' AND x.candidate_id=$1) GROUP BY x.status`, u.ID, u.Role)
		if err != nil {
			return nil, internal(err)
		}
		for rows.Next() {
			var s string
			var n int
			if err = rows.Scan(&s, &n); err != nil {
				rows.Close()
				return nil, internal(err)
			}
			out.Applications[s] = n
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return nil, internal(err)
		}
		if u.Role == "employer" {
			out.Categories, err = jsonRows[CategoryCount](ctx, a, `SELECT row_to_json(t) FROM (SELECT specialization,grade,count(*) AS count FROM profiles p JOIN users u ON u.id=p.user_id WHERE p.published AND p.grade<>'' AND u.verified GROUP BY specialization,grade ORDER BY specialization,grade) t`)
			if err != nil {
				return nil, internal(err)
			}
			for _, c := range out.Categories {
				out.CandidateCount += c.Count
			}
			err = a.DB.QueryRow(ctx, "SELECT (SELECT count(*) FROM vacancies WHERE employer_id=$1 AND status='published'),(SELECT count(*) FROM shortlist WHERE employer_id=$1)", u.ID).Scan(&out.PublishedVacancies, &out.SavedCandidates)
			if err != nil {
				return nil, internal(err)
			}
		}
		return ok(out), nil
	})
}
