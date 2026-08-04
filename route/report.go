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

		var report models.Report

		err := ctx.BindJSON(&report)
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

		if report.ServiceID == "" {
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

		saved, err := db.SaveReport(database, report)
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
