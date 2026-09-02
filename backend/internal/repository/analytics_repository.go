package repository

import (
	"context"
	"fmt"

	"ai-ats-platform/backend/internal/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AnalyticsRepository struct {
	pool *pgxpool.Pool
}

func NewAnalyticsRepository(pool *pgxpool.Pool) *AnalyticsRepository {
	return &AnalyticsRepository{pool: pool}
}

func (r *AnalyticsRepository) Overview(ctx context.Context, companyID uuid.UUID) (*domain.AnalyticsOverview, error) {
	out := &domain.AnalyticsOverview{ByStatus: map[string]int64{}}

	if err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*),
		       COUNT(*) FILTER (WHERE status = 'open'),
		       COUNT(*) FILTER (WHERE status = 'closed')
		FROM jobs WHERE company_id = $1
	`, companyID).Scan(&out.TotalJobs, &out.OpenJobs, &out.ClosedJobs); err != nil {
		return nil, err
	}

	// Total applications should derive from the same candidate dataset used for status counts.
	// This keeps candidate counts, status distribution, and application totals consistent for a company.

	rows, err := r.pool.Query(ctx, `
		SELECT status, COUNT(*) FROM candidates WHERE company_id = $1 GROUP BY status
	`, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var status string
		var count int64
		if err := rows.Scan(&status, &count); err != nil {
			return nil, err
		}
		out.ByStatus[status] = count
		out.Applicants += count
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Use the candidate status counts as the source of truth for total applications.
	out.Applications = out.Applicants

	out.AIShortlisted = out.ByStatus["ai_shortlisted"]
	out.RecruiterShortlisted = out.ByStatus["recruiter_shortlisted"] + out.ByStatus["shortlisted"]
	out.Interviews = out.ByStatus["interview"]
	out.Offers = out.ByStatus["offer"]
	out.Selected = out.ByStatus["selected"]
	out.Rejected = out.ByStatus["rejected"]
	out.Hired = out.ByStatus["hired"]

	var avg *float64
	if err := r.pool.QueryRow(ctx, `
		SELECT AVG(overall_score)::float8 FROM candidates
		WHERE company_id = $1 AND overall_score IS NOT NULL
	`, companyID).Scan(&avg); err != nil {
		return nil, err
	}
	out.AvgAIMatch = avg

	var avgHire *float64
	_ = r.pool.QueryRow(ctx, `
		SELECT AVG(EXTRACT(EPOCH FROM (updated_at - created_at)) / 86400.0)::float8
		FROM candidates
		WHERE company_id = $1 AND status IN ('hired', 'selected')
	`, companyID).Scan(&avgHire)
	out.AvgTimeToHireDays = avgHire

	offers := out.Offers + out.Hired
	if offers > 0 {
		rate := float64(out.Hired) / float64(offers) * 100
		out.OfferAcceptanceRate = &rate
	}

	out.Funnel = []domain.NamedCount{
		{Name: "Applied", Count: out.ByStatus["applied"] + out.ByStatus["screening"]},
		{Name: "AI Shortlisted", Count: out.AIShortlisted},
		{Name: "Recruiter Shortlisted", Count: out.RecruiterShortlisted},
		{Name: "Interview", Count: out.Interviews},
		{Name: "Offer", Count: out.Offers},
		{Name: "Hired/Selected", Count: out.Hired + out.Selected},
		{Name: "Rejected", Count: out.Rejected},
	}

	out.ApplicationsPerJob, err = r.namedCounts(ctx, `
		SELECT COALESCE(j.title, 'Unassigned'), COUNT(a.id)
		FROM applications a
		LEFT JOIN jobs j ON j.id = a.job_id
		WHERE j.company_id = $1 AND a.deleted_at IS NULL
		GROUP BY COALESCE(j.title, 'Unassigned')
		ORDER BY COUNT(a.id) DESC
		LIMIT 12
	`, companyID)
	if err != nil {
		return nil, err
	}

	out.TopSkills, err = r.namedCounts(ctx, `
		SELECT skill, COUNT(*) FROM (
			SELECT UNNEST(skills) AS skill FROM candidates WHERE company_id = $1
		) t GROUP BY skill ORDER BY COUNT(*) DESC LIMIT 15
	`, companyID)
	if err != nil {
		return nil, err
	}

	out.MissingSkills, err = r.namedCounts(ctx, `
		SELECT skill, COUNT(*) FROM (
			SELECT UNNEST(missing_skills) AS skill FROM candidates WHERE company_id = $1
		) t GROUP BY skill ORDER BY COUNT(*) DESC LIMIT 15
	`, companyID)
	if err != nil {
		return nil, err
	}

	out.AIMatchDistribution, err = r.buckets(ctx, companyID)
	if err != nil {
		return nil, err
	}

	out.HiringTrend, err = r.trend(ctx, `
		SELECT to_char(date_trunc('week', created_at), 'YYYY-MM-DD'), COUNT(*)
		FROM candidates WHERE company_id = $1 AND created_at >= NOW() - INTERVAL '12 weeks'
		GROUP BY 1 ORDER BY 1
	`, companyID)
	if err != nil {
		return nil, err
	}

	out.MonthlyHiring, err = r.trend(ctx, `
		SELECT to_char(date_trunc('month', updated_at), 'YYYY-MM'), COUNT(*)
		FROM candidates
		WHERE company_id = $1 AND status IN ('hired', 'selected')
		  AND updated_at >= NOW() - INTERVAL '12 months'
		GROUP BY 1 ORDER BY 1
	`, companyID)
	if err != nil {
		return nil, err
	}

	out.RecruiterProductivity, err = r.namedCounts(ctx, `
		SELECT COALESCE(u.email, 'unassigned'), COUNT(c.id)
		FROM candidates c
		LEFT JOIN users u ON u.id = c.assigned_to
		WHERE c.company_id = $1
		GROUP BY u.email
		ORDER BY COUNT(c.id) DESC
		LIMIT 10
	`, companyID)
	if err != nil {
		return nil, fmt.Errorf("recruiter productivity: %w", err)
	}

	return out, nil
}

func (r *AnalyticsRepository) namedCounts(ctx context.Context, sql string, companyID uuid.UUID) ([]domain.NamedCount, error) {
	rows, err := r.pool.Query(ctx, sql, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]domain.NamedCount, 0)
	for rows.Next() {
		var n domain.NamedCount
		if err := rows.Scan(&n.Name, &n.Count); err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

func (r *AnalyticsRepository) trend(ctx context.Context, sql string, companyID uuid.UUID) ([]domain.TrendPoint, error) {
	rows, err := r.pool.Query(ctx, sql, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]domain.TrendPoint, 0)
	for rows.Next() {
		var t domain.TrendPoint
		if err := rows.Scan(&t.Period, &t.Count); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (r *AnalyticsRepository) buckets(ctx context.Context, companyID uuid.UUID) ([]domain.BucketCount, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT bucket, COUNT(*) FROM (
			SELECT CASE
				WHEN overall_score IS NULL THEN 'Unscored'
				WHEN overall_score < 40 THEN '0-39'
				WHEN overall_score < 55 THEN '40-54'
				WHEN overall_score < 70 THEN '55-69'
				WHEN overall_score < 85 THEN '70-84'
				ELSE '85-100'
			END AS bucket
			FROM candidates WHERE company_id = $1
		) t GROUP BY bucket
	`, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]domain.BucketCount, 0)
	for rows.Next() {
		var b domain.BucketCount
		if err := rows.Scan(&b.Bucket, &b.Count); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}
