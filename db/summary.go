package db

import (
	"chronolens/models"
	"database/sql"
)

// solverSummarySQL agregira statistiku po solveru. $1 je opcioni filter po function koloni (prazan string znači
// bez filtera, prati konvenciju iz GetResults gde prazan parametar odgovara svemu). solved_rate ide preko SVIH
// redova solvera; avg_steps/avg_value idu SAMO preko status='solved' redova i prirodno vraćaju NULL kad takvih
// redova nema (AVG preko praznog skupa). win_rate se računa preko batch_id IS NOT NULL redova: za svaki batch
// nalazi min(value) među status='solved' redovima tog batch-a, solver "pobeđuje" ako je status='solved' i
// |value-min| < 1e-9 (dozvoljava remije), win_rate = win_count/participations, ili NULL ako solver nikad nije bio
// deo batch-a (participations=0)
const solverSummarySQL = `
WITH batch_min AS (
	SELECT batch_id, MIN(value) AS min_value
	FROM reports
	WHERE batch_id IS NOT NULL AND status = 'solved' AND (function = $1 OR $1 = '')
	GROUP BY batch_id
),
wins AS (
	SELECT r.method, COUNT(*) AS win_count
	FROM reports r
	JOIN batch_min bm ON r.batch_id = bm.batch_id
	WHERE r.status = 'solved' AND ABS(r.value - bm.min_value) < 1e-9 AND (r.function = $1 OR $1 = '')
	GROUP BY r.method
),
participations AS (
	SELECT method, COUNT(DISTINCT batch_id) AS participations
	FROM reports
	WHERE batch_id IS NOT NULL AND (function = $1 OR $1 = '')
	GROUP BY method
),
base AS (
	SELECT
		COALESCE(method, '') AS method,
		COUNT(*) AS total_runs,
		COUNT(*) FILTER (WHERE status = 'solved')::float / COUNT(*) AS solved_rate,
		AVG(steps) FILTER (WHERE status = 'solved') AS avg_steps,
		AVG(value) FILTER (WHERE status = 'solved') AS avg_value
	FROM reports
	WHERE (function = $1 OR $1 = '')
	GROUP BY method
)
SELECT
	b.method,
	b.total_runs,
	b.solved_rate,
	COALESCE(p.participations, 0) AS participations,
	COALESCE(w.win_count, 0) AS win_count,
	CASE WHEN COALESCE(p.participations, 0) > 0
	     THEN COALESCE(w.win_count, 0)::float / p.participations
	     ELSE NULL END AS win_rate,
	b.avg_steps,
	b.avg_value
FROM base b
LEFT JOIN participations p ON b.method = p.method
LEFT JOIN wins w ON b.method = w.method
ORDER BY b.method
`

// GetSolverSummary vraća agregiranu statistiku po solveru, opciono filtriranu na jednu function (prazan string
// = sve funkcije)
func GetSolverSummary(database *sql.DB, functionFilter string) ([]models.SolverSummary, error) {

	rows, err := database.Query(solverSummarySQL, functionFilter)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	summaries := make([]models.SolverSummary, 0)
	for rows.Next() {
		var s models.SolverSummary
		var winRate, avgSteps, avgValue sql.NullFloat64

		if err := rows.Scan(
			&s.Method,
			&s.TotalRuns,
			&s.SolvedRate,
			&s.Participations,
			&s.WinCount,
			&winRate,
			&avgSteps,
			&avgValue,
		); err != nil {
			return nil, err
		}

		if winRate.Valid {
			v := winRate.Float64
			s.WinRate = &v
		}
		if avgSteps.Valid {
			v := avgSteps.Float64
			s.AvgSteps = &v
		}
		if avgValue.Valid {
			v := avgValue.Float64
			s.AvgValue = &v
		}

		summaries = append(summaries, s)
	}

	return summaries, rows.Err()
}

// functionSummarySQL radi isti win_rate/avg_value proračun kao solverSummarySQL, ali grupisano po (function,
// method) da bi se za svaku funkciju izabrao najbolji solver po svakom kriterijumu nezavisno. Funkcije bez
// ijednog batch_id reda dobijaju best_solver_by_win_rate=NULL (win_rates CTE ih nikad ne generiše jer
// participations>0 uslov ne prolazi); best_solver_by_avg_value se računa nezavisno od batch_id
const functionSummarySQL = `
WITH batch_min AS (
	SELECT batch_id, function, MIN(value) AS min_value
	FROM reports
	WHERE batch_id IS NOT NULL AND status = 'solved'
	GROUP BY batch_id, function
),
wins AS (
	SELECT r.function, r.method, COUNT(*) AS win_count
	FROM reports r
	JOIN batch_min bm ON r.batch_id = bm.batch_id AND r.function = bm.function
	WHERE r.status = 'solved' AND ABS(r.value - bm.min_value) < 1e-9
	GROUP BY r.function, r.method
),
participations AS (
	SELECT function, method, COUNT(DISTINCT batch_id) AS participations
	FROM reports
	WHERE batch_id IS NOT NULL
	GROUP BY function, method
),
win_rates AS (
	SELECT p.function, p.method,
	       COALESCE(w.win_count, 0)::float / p.participations AS win_rate
	FROM participations p
	LEFT JOIN wins w ON p.function = w.function AND p.method = w.method
	WHERE p.participations > 0
),
best_win AS (
	SELECT DISTINCT ON (function) function, method AS best_method
	FROM win_rates
	ORDER BY function, win_rate DESC, method ASC
),
avg_values AS (
	SELECT function, method, AVG(value) AS avg_value
	FROM reports
	WHERE status = 'solved'
	GROUP BY function, method
),
best_avg AS (
	SELECT DISTINCT ON (function) function, method AS best_method
	FROM avg_values
	ORDER BY function, avg_value ASC, method ASC
),
totals AS (
	SELECT function, COUNT(*) AS total_runs
	FROM reports
	WHERE function IS NOT NULL AND function <> ''
	GROUP BY function
)
SELECT
	t.function,
	t.total_runs,
	bw.best_method,
	ba.best_method
FROM totals t
LEFT JOIN best_win bw ON t.function = bw.function
LEFT JOIN best_avg ba ON t.function = ba.function
ORDER BY t.function
`

// GetFunctionSummary vraća agregiranu statistiku po benchmark funkciji preko svih sačuvanih izveštaja
func GetFunctionSummary(database *sql.DB) ([]models.FunctionSummary, error) {

	rows, err := database.Query(functionSummarySQL)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	summaries := make([]models.FunctionSummary, 0)
	for rows.Next() {
		var f models.FunctionSummary
		var bestWinRate, bestAvgValue sql.NullString

		if err := rows.Scan(
			&f.Function,
			&f.TotalRuns,
			&bestWinRate,
			&bestAvgValue,
		); err != nil {
			return nil, err
		}

		if bestWinRate.Valid {
			v := bestWinRate.String
			f.BestSolverByWinRate = &v
		}
		if bestAvgValue.Valid {
			v := bestAvgValue.String
			f.BestSolverByAvgValue = &v
		}

		summaries = append(summaries, f)
	}

	return summaries, rows.Err()
}

// GetTotalReportCount vraća ukupan broj sačuvanih izveštaja preko cele reports tabele
func GetTotalReportCount(database *sql.DB) (int, error) {

	var total int
	err := database.QueryRow(`SELECT COUNT(*) FROM reports`).Scan(&total)

	return total, err
}
