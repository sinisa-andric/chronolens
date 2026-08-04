package db

import (
	"chronolens/models"
	"database/sql"

	"github.com/lib/pq"
)

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

	return nil
}

// SaveReport ubacuje izveštaj i vraća ga popunjenog sa generisanim id-jem i vremenskom oznakom
func SaveReport(database *sql.DB, report models.Report) (models.Report, error) {

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

	err := database.QueryRow(
		`INSERT INTO reports (service_id, method, point, value, steps, status, error, duration_ms, function, batch_id, received_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW())
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
	).Scan(&report.ID, &report.ReceivedAt)

	return report, err
}

// GetResults vraća do limita izveštaja, najnovije prvo, opciono filtrirane po method-u, status-u i/ili function-u (prazan filter odgovara svemu)
func GetResults(database *sql.DB, method, status, function string, limit int) ([]models.Report, error) {

	rows, err := database.Query(
		`SELECT id, service_id, method, point, value, steps, status, error, duration_ms, function, received_at
		 FROM reports
		 WHERE (method = $1 OR $1 = '')
		 AND (status = $2 OR $2 = '')
		 AND (function = $3 OR $3 = '')
		 ORDER BY id DESC
		 LIMIT $4`,
		method,
		status,
		function,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := make([]models.Report, 0, limit)
	for rows.Next() {
		var report models.Report
		var status, errStr, function *string
		var durationMs *int64

		if err := rows.Scan(
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
		); err != nil {
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

		results = append(results, report)
	}

	return results, rows.Err()
}
