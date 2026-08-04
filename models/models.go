package models

import "time"

// Report je rezultat primljen od nadgledanog servisa, sačuvan radi uvida
type Report struct {
	ID         int64     `json:"id,omitempty"`
	ServiceID  string    `json:"service_id"`
	Method     string    `json:"method,omitempty"`
	Point      []float64 `json:"point,omitempty"`
	Value      float64   `json:"value,omitempty"`
	Steps      int       `json:"steps,omitempty"`
	Status     string    `json:"status,omitempty"`
	Error      string    `json:"error,omitempty"`
	DurationMs int64     `json:"duration_ms,omitempty"`
	Function   string    `json:"function,omitempty"`
	BatchID    string    `json:"batch_id,omitempty"`
	ReceivedAt time.Time `json:"received_at,omitempty"`
}

// SolverSummary agregira statistiku jednog solvera preko svih sačuvanih izveštaja
type SolverSummary struct {
	Method         string   `json:"method"`
	TotalRuns      int      `json:"total_runs"`
	SolvedRate     float64  `json:"solved_rate"`
	Participations int      `json:"participations"`
	WinCount       int      `json:"win_count"`
	WinRate        *float64 `json:"win_rate"`
	AvgSteps       *float64 `json:"avg_steps"`
	AvgValue       *float64 `json:"avg_value"`
}

// FunctionSummary agregira statistiku jedne benchmark funkcije preko svih sačuvanih izveštaja
type FunctionSummary struct {
	Function             string  `json:"function"`
	TotalRuns            int     `json:"total_runs"`
	BestSolverByWinRate  *string `json:"best_solver_by_win_rate"`
	BestSolverByAvgValue *string `json:"best_solver_by_avg_value"`
}
