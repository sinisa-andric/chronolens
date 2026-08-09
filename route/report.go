package route

import (
	"chronolens/db"
	"chronolens/models"
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type ChronolensResponse struct {
	Approved bool   `json:"approved"`
	Message  string `json:"message,omitempty"`
	Response any    `json:"response,omitempty"`
}

// reportRequest je models.Report dopunjen opcionom putanjom pretrage u telu zahteva — embedding
// umesto dodavanja polja direktno u models.Report da bi trajno čuvanje putanje ostalo ograničeno
// na db i route pakete
type reportRequest struct {
	models.Report
	Trajectory []db.TrajectoryPoint `json:"trajectory,omitempty"`
}

// ReportHandler prima prijavljene rezultate i trajno ih čuva
func ReportHandler(database *sql.DB) gin.HandlerFunc {

	return func(ctx *gin.Context) {

		log := zap.New(nil)

		defer log.Sync() // Flush logs before exiting

		requestId := ctx.GetString("id")
		log = log.Named("[Chronolens:ReportHandler]").WithOptions(
			zap.Fields(
				zap.String("requestId", requestId),
			),
		)

		log.Info("report received")

		var req reportRequest

		err := ctx.BindJSON(&req)
		if err != nil {
			log.Error("failed to bind report body", zap.Error(err))
			ctx.JSON(
				http.StatusBadRequest,
				ChronolensResponse{
					Approved: false,
					Message:  "failed to bind report body: " + err.Error(),
					Response: nil,
				},
			)
			return
		}

		if req.ServiceID == "" {
			ctx.JSON(
				http.StatusBadRequest,
				ChronolensResponse{
					Approved: false,
					Message:  "service_id is required",
					Response: nil,
				},
			)
			return
		}

		saved, err := db.SaveReport(database, req.Report, req.Trajectory)
		if err != nil {
			log.Error("failed to save report", zap.Error(err))
			ctx.JSON(
				http.StatusInternalServerError,
				ChronolensResponse{
					Approved: false,
					Message:  "failed to save report: " + err.Error(),
					Response: nil,
				},
			)
			return
		}

		ctx.JSON(
			http.StatusCreated,
			ChronolensResponse{
				Approved: true,
				Message:  "report saved",
				Response: saved,
			},
		)

		log.Info("report saved",
			zap.Int64("id", saved.ID),
			zap.String("serviceId", saved.ServiceID),
		)
	}
}
