package main

import (
	"os"

	"github.com/gin-gonic/gin"
	"github.com/nats-io/nats.go"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
	"internal/activities"
	"github.com/Namanv0509/BankTransfer-Temporal-Workflow/internal/api"
	"github.com/Namanv0509/BankTransfer-Temporal-Workflow/internal/config"
	"github.com/Namanv0509/BankTransfer-Temporal-Workflow/internal/models"
	"github.com/Namanv0509/BankTransfer-Temporal-Workflow/internal/observability"
	"github.com/Namanv0509/BankTransfer-Temporal-Workflow/internal/workflows"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

func main() {

	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()

	cfg, err := config.LoadConfig()
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to load config")
	}


	db, err := models.NewDB(cfg.PostgresDSN)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to initialize database")
	}

	nc, err := nats.Connect(cfg.NATSHost)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to connect to NATS")
	}
	defer nc.Close()

	tc, err := client.Dial(client.Options{
		HostPort: cfg.TemporalHost,
	})
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to create Temporal client")
	}
	defer tc.Close()

	metrics := observability.NewMetrics()

	w := worker.New(tc, "meshery-transfer-queue", worker.Options{})
	activityCtx := &activities.ActivityContext{
		DB:      db,
		NATS:    nc,
		Logger:  &logger,
		Metrics: metrics,
	}


	w.RegisterWorkflowWithOptions(
		func(ctx workflow.Context, req models.TransferRequest) (models.TransferResult, error) {
			return workflows.InterbankTransferWorkflow(ctx, req, metrics)
		},
		workflow.RegisterOptions{Name: "InterbankTransferWorkflow"},
	)
	w.RegisterActivityWithOptions(activityCtx.ValidateAccount, activity.RegisterOptions{Name: "ValidateAccount"})
	w.RegisterActivityWithOptions(activityCtx.WithdrawFunds, activity.RegisterOptions{Name: "WithdrawFunds"})
	w.RegisterActivityWithOptions(activityCtx.DepositFunds, activity.RegisterOptions{Name: "DepositFunds"})
	w.RegisterActivityWithOptions(activityCtx.LogTransaction, activity.RegisterOptions{Name: "LogTransaction"})
	w.RegisterActivityWithOptions(activityCtx.PublishEvent, activity.RegisterOptions{Name: "PublishEvent"})


	go func() {
		if err := w.Run(worker.InterruptCh()); err != nil {
			logger.Fatal().Err(err).Msg("Failed to start worker")
		}
	}()


	r := gin.Default()
	r.POST("/transfer", func(c *gin.Context) { api.StartTransfer(c, tc, &logger) })
	r.GET("/status/:workflowID", func(c *gin.Context) { api.GetStatus(c, tc) })
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))
	if err := r.Run(":" + cfg.HTTPPort); err != nil {
		logger.Fatal().Err(err).Msg("Failed to start HTTP server")
	}
}