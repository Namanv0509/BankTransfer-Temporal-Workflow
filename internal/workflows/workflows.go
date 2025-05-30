package workflows

import (
	"time"

	"BankTransfer-Temporal-Workflow/internal/models"
	"BankTransfer-Temporal-Workflow/internal/observability"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

func InterbankTransferWorkflow(ctx workflow.Context, req models.TransferRequest, metrics *observability.Metrics) (models.TransferResult, error) {
	startTime := time.Now()
	logger := workflow.GetLogger(ctx)
	result := models.TransferResult{Status: "Failed"}

	ao := workflow.ActivityOptions{
		StartToCloseTimeout: time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumAttempts:    3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	err := workflow.ExecuteActivity(ctx, "PublishEvent", workflow.GetInfo(ctx).WorkflowExecution.ID, "Initiated").Get(ctx, nil)
	if err != nil {
		logger.Error("Failed to publish start event", "error", err)
	}

	if err := workflow.ExecuteActivity(ctx, "ValidateAccount", req).Get(ctx, nil); err != nil {
		logger.Error("Account validation failed", "error", err)
		workflow.ExecuteActivity(ctx, "PublishEvent", workflow.GetInfo(ctx).WorkflowExecution.ID, "Failed")
		return result, err
	}

	if err := workflow.ExecuteActivity(ctx, "WithdrawFunds", req).Get(ctx, nil); err != nil {
		logger.Error("Withdraw funds failed", "error", err)
		workflow.ExecuteActivity(ctx, "PublishEvent", workflow.GetInfo(ctx).WorkflowExecution.ID, "Failed")
		return result, err
	}

	if err := workflow.ExecuteActivity(ctx, "DepositFunds", req).Get(ctx, nil); err != nil {
		logger.Error("Deposit funds failed", "error", err)
		workflow.ExecuteActivity(ctx, "DepositFunds", models.TransferRequest{
			FromAccountID: req.ToAccountID,
			ToAccountID:   req.FromAccountID,
			Amount:        req.Amount,
		})
		workflow.ExecuteActivity(ctx, "PublishEvent", workflow.GetInfo(ctx).WorkflowExecution.ID, "Failed")
		return result, err
	}

	var txID string
	if err := workflow.ExecuteActivity(ctx, "LogTransaction", req).Get(ctx, &txID); err != nil {
		logger.Error("Log transaction failed", "error", err)
		workflow.ExecuteActivity(ctx, "PublishEvent", workflow.GetInfo(ctx).WorkflowExecution.ID, "Failed")
		return result, err
	}

	result.TransactionID = txID
	result.Status = "Completed"
	workflow.ExecuteActivity(ctx, "PublishEvent", workflow.GetInfo(ctx).WorkflowExecution.ID, "Completed")

	duration := time.Since(startTime).Seconds()
	metrics.WorkflowDuration.WithLabelValues("InterbankTransferWorkflow", result.Status).Observe(duration)

	return result, nil
}
