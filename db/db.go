package db

import (
	"chronolens/models"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/lib/pq"
)

// TrajectoryPoint je jedna zabeležena tačka putanje pretrage — isti format koji algorithmia/problemia
// već koriste ({step, point, value}), sačuvan ovde umesto u chronolens/models da bi trajno čuvanje
// putanje ostalo ograničeno na db i route pakete
type TrajectoryPoint struct {
	Step  int       `json:"step"`
	Point []float64 `json:"point"`
	Value float64   `json:"value"`
}

// ReportWithTrajectory je models.Report dopunjen opcionom putanjom — Trajectory ostaje nil/omitempty
// kad trajectory kolona u bazi nije tražena (include_trajectory=false) ili je NULL za taj red, tako
// da je JSON izlaz identičan običnom models.Report u tim slučajevima
type ReportWithTrajectory struct {
	models.Report
	Trajectory []TrajectoryPoint `json:"trajectory,omitempty"`
}

const createReportsTableSQL = `
CREATE TABLE IF NOT EXISTS reports (
	id SERIAL PRIMARY KEY,
	service_id TEXT NOT NULL,
	method TEXT,
	point DOUBLE PRECISION[],
	value DOUBLE PRECISION,
	steps INTEGER,
	status TEXT NOT NULL DEFAULT 'solved',
	error TEXT NOT NULL DEFAULT '',
	duration_ms BIGINT NOT NULL DEFAULT 0,
	received_at TIMESTAMPTZ NOT NULL DEFAULT now()
)`

// EnsureReportsTable kreira tabelu reports ako već ne postoji
func EnsureReportsTable(database *sql.DB) error {

	_, err := database.Exec(createReportsTableSQL)
	if err != nil {
		return err
	}

	_, err = database.Exec(`ALTER TABLE reports ADD COLUMN IF NOT EXISTS status TEXT`)
	if err != nil {
		return err
	}
	_, err = database.Exec(`ALTER TABLE reports ADD COLUMN IF NOT EXISTS error TEXT`)
	if err != nil {
		return err
	}
	_, err = database.Exec(`ALTER TABLE reports ADD COLUMN IF NOT EXISTS duration_ms BIGINT`)
	if err != nil {
		return err
	}
	_, err = database.Exec(`ALTER TABLE reports ADD COLUMN IF NOT EXISTS function TEXT`)
	if err != nil {
		return err
	}
	_, err = database.Exec(`ALTER TABLE reports ADD COLUMN IF NOT EXISTS batch_id VARCHAR(64)`)
	if err != nil {
		return err
	}
	_, err = database.Exec(`CREATE INDEX IF NOT EXISTS idx_reports_batch_id ON reports(batch_id)`)
	if err != nil {
		return err
	}
	// nullable, bez indeksa — ne pretražujemo po sadržaju trajectory-ja, samo ga trajno čuvamo
	_, err = database.Exec(`ALTER TABLE reports ADD COLUMN IF NOT EXISTS trajectory JSONB`)
	if err != nil {
		return err
	}

	return nil
}

// SaveReport ubacuje izveštaj (opciono sa putanjom pretrage) i vraća ga popunjenog sa generisanim
// id-jem i vremenskom oznakom
func SaveReport(database *sql.DB, report models.Report, trajectory []TrajectoryPoint) (models.Report, error) {

	if report.Status == "" {
		report.Status = "solved"
	}

	// prazan batch_id (stariji klijenti koji ne šalju to polje) mora biti sačuvan kao SQL NULL, ne kao prazan
	// string — u suprotnom bi svi takvi izveštaji delili istu (praznu) "batch" vrednost i lažno se grupisali
	// zajedno u win_rate proračunu koji filtrira po batch_id IS NOT NULL
	var batchID *string
	if report.BatchID != "" {
		batchID = &report.BatchID
	}

	// isti razlog kao batch_id — solveri koji nisu tražili include_trajectory ne smeju upisati prazan
	// niz, mora biti SQL NULL da se razlikuje od "eksplicitno prazna putanja". Prosleđeno kao *string
	// (ne []byte) jer lib/pq kodira []byte parametre kao bytea binarni format, što postgres odbija
	// za jsonb kolonu ("invalid input syntax for type json") — tekstualni parametar radi ispravno.
	var trajectoryParam *string
	if len(trajectory) > 0 {
		trajectoryJSON, err := json.Marshal(trajectory)
		if err != nil {
			return report, fmt.Errorf("failed to marshal trajectory: %w", err)
		}
		s := string(trajectoryJSON)
		trajectoryParam = &s
	}

	err := database.QueryRow(
		`INSERT INTO reports (service_id, method, point, value, steps, status, error, duration_ms, function, batch_id, trajectory, received_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW())
		 RETURNING id, received_at`,
		report.ServiceID,
		report.Method,
		pq.Array(report.Point),
		report.Value,
		report.Steps,
		report.Status,
		report.Error,
		report.DurationMs,
		report.Function,
		batchID,
		trajectoryParam,
	).Scan(&report.ID, &report.ReceivedAt)

	return report, err
}

// GetResults vraća do limita izveštaja, najnovije prvo, opciono filtrirane po method-u, status-u i/ili
// function-u (prazan filter odgovara svemu). Kad je includeTrajectory false, trajectory kolona se ne
// selektuje niti dira — identično ponašanje kao pre nego što je trajno čuvanje putanje dodato, da se
// ne opterećuje upit kad se putanja ne traži.
func GetResults(database *sql.DB, method, status, function string, limit int, includeTrajectory bool) ([]ReportWithTrajectory, error) {

	columns := "id, service_id, method, point, value, steps, status, error, duration_ms, function, received_at"
	if includeTrajectory {
		columns += ", trajectory"
	}

	rows, err := database.Query(
		fmt.Sprintf(
			`SELECT %s
			 FROM reports
			 WHERE (method = $1 OR $1 = '')
			 AND (status = $2 OR $2 = '')
			 AND (function = $3 OR $3 = '')
			 ORDER BY id DESC
			 LIMIT $4`,
			columns,
		),
		method,
		status,
		function,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := make([]ReportWithTrajectory, 0, limit)
	for rows.Next() {
		var report models.Report
		var status, errStr, function *string
		var durationMs *int64
		var trajectoryRaw []byte

		scanArgs := []any{
			&report.ID,
			&report.ServiceID,
			&report.Method,
			pq.Array(&report.Point),
			&report.Value,
			&report.Steps,
			&status,
			&errStr,
			&durationMs,
			&function,
			&report.ReceivedAt,
		}
		if includeTrajectory {
			scanArgs = append(scanArgs, &trajectoryRaw)
		}

		if err := rows.Scan(scanArgs...); err != nil {
			return nil, err
		}

		if status != nil {
			report.Status = *status
		}
		if errStr != nil {
			report.Error = *errStr
		}
		if durationMs != nil {
			report.DurationMs = *durationMs
		}
		if function != nil {
			report.Function = *function
		}

		item := ReportWithTrajectory{Report: report}
		// trajectoryRaw je nil za redove gde je kolona SQL NULL (stari red ili taj poziv nije tražio
		// putanju) — polje ostaje nil/omitempty, ne pokušavamo unmarshal na NULL
		if includeTrajectory && trajectoryRaw != nil {
			if err := json.Unmarshal(trajectoryRaw, &item.Trajectory); err != nil {
				return nil, fmt.Errorf("failed to unmarshal trajectory for report %d: %w", report.ID, err)
			}
		}

		results = append(results, item)
	}

	return results, rows.Err()
}
