package route

import (
	"chronolens/db"
	"database/sql"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

const (
	resultsDefaultLimit = 20
	resultsMaxLimit     = 100
)

// ResultsResponse sadrži sačuvane izveštaje koji odgovaraju traženim filterima. Trajectory polje
// svakog izveštaja je omitempty — odsutno osim kad je include_trajectory=true I ta kolona nije NULL
// za taj red, tako da je JSON oblik identičan pre-trajectory ponašanju kad se ne traži.
type ResultsResponse struct {
	Total   int                       `json:"total"`
	Results []db.ReportWithTrajectory `json:"results"`
}

// ResultsHandler vraća sačuvane izveštaje, opciono filtrirane po method-u i/ili status-u, najnovije prvo
func ResultsHandler(database *sql.DB) gin.HandlerFunc {

	return func(ctx *gin.Context) {

		log := zap.New(nil)

		defer log.Sync() // Flush logs before exiting

		requestId := ctx.GetString("id")
		log = log.Named("[Chronolens:ResultsHandler]").WithOptions(
			zap.Fields(
				zap.String("requestId", requestId),
			),
		)

		log.Info("results requested")

		method := ctx.Query("method")
		status := ctx.Query("status")
		functionFilter := ctx.Query("function")
		includeTrajectory := ctx.Query("include_trajectory") == "true"

		limit := resultsDefaultLimit
		if raw := ctx.Query("limit"); raw != "" {
			if parsed, err := strconv.Atoi(raw); err == nil {
				limit = parsed
			}
		}
		if limit > resultsMaxLimit {
			limit = resultsMaxLimit
		}
		if limit < 1 {
			limit = 1
		}

		results, err := db.GetResults(database, method, status, functionFilter, limit, includeTrajectory)
		if err != nil {
			log.Error("failed to fetch results", zap.Error(err))
			ctx.JSON(
				http.StatusInternalServerError,
				ChronolensResponse{
					Approved: false,
					Message:  "failed to fetch results: " + err.Error(),
					Response: nil,
				},
			)
			return
		}

		response := ResultsResponse{
			Total:   len(results),
			Results: results,
		}

		ctx.JSON(
			http.StatusOK,
			ChronolensResponse{
				Approved: true,
				Message:  "results fetched",
				Response: response,
			},
		)

		log.Info("results fetched",
			zap.Int("total", response.Total),
			zap.String("method", method),
			zap.String("status", status),
		)
	}
}
