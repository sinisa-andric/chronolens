package route

import (
	"chronolens/db"
	"chronolens/models"
	"database/sql"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// SummaryResponse sadrži agregiranu statistiku po solveru i po funkciji preko svih sačuvanih izveštaja
type SummaryResponse struct {
	GeneratedAt string                   `json:"generated_at"`
	TotalRuns   int                      `json:"total_runs"`
	PerSolver   []models.SolverSummary   `json:"per_solver"`
	PerFunction []models.FunctionSummary `json:"per_function"`
}

// SummaryHandler vraća agregiranu statistiku (solved_rate, win_rate, avg_steps, avg_value) po solveru i po
// benchmark funkciji preko svih sačuvanih izveštaja. Opcioni query parametar ?function= filtrira per_solver
// deo odgovora na tu funkciju; per_function deo uvek pokriva sve funkcije
func SummaryHandler(database *sql.DB) gin.HandlerFunc {

	return func(ctx *gin.Context) {

		log := zap.New(nil)

		defer log.Sync() // Flush logs before exiting

		requestId := ctx.GetString("id")
		log = log.Named("[Chronolens:SummaryHandler]").WithOptions(
			zap.Fields(
				zap.String("requestId", requestId),
			),
		)

		log.Info("summary requested")

		functionFilter := ctx.Query("function")

		totalRuns, err := db.GetTotalReportCount(database)
		if err != nil {
			log.Error("failed to count reports", zap.Error(err))
			ctx.JSON(
				http.StatusInternalServerError,
				ChronolensResponse{
					Approved: false,
					Message:  "failed to count reports: " + err.Error(),
					Response: nil,
				},
			)
			return
		}

		perSolver, err := db.GetSolverSummary(database, functionFilter)
		if err != nil {
			log.Error("failed to fetch solver summary", zap.Error(err))
			ctx.JSON(
				http.StatusInternalServerError,
				ChronolensResponse{
					Approved: false,
					Message:  "failed to fetch solver summary: " + err.Error(),
					Response: nil,
				},
			)
			return
		}

		perFunction, err := db.GetFunctionSummary(database)
		if err != nil {
			log.Error("failed to fetch function summary", zap.Error(err))
			ctx.JSON(
				http.StatusInternalServerError,
				ChronolensResponse{
					Approved: false,
					Message:  "failed to fetch function summary: " + err.Error(),
					Response: nil,
				},
			)
			return
		}

		response := SummaryResponse{
			GeneratedAt: time.Now().UTC().Format(time.RFC3339),
			TotalRuns:   totalRuns,
			PerSolver:   perSolver,
			PerFunction: perFunction,
		}

		ctx.JSON(
			http.StatusOK,
			ChronolensResponse{
				Approved: true,
				Message:  "summary generated",
				Response: response,
			},
		)

		log.Info("summary generated",
			zap.Int("totalRuns", totalRuns),
			zap.Int("solvers", len(perSolver)),
			zap.Int("functions", len(perFunction)),
			zap.String("functionFilter", functionFilter),
		)
	}
}
