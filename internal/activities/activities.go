package activities

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog"
	"github.com/Namanv0509/BankTransfer-Temporal-Workflow/internal/models"
	"github.com/Namanv0509/BankTransfer-Temporal-Workflow/internal/observability"
	"gorm.io/gorm"
)

type ActivityContext struct {
	DB      *models.DB
	NATS    *nats.Conn
	Logger  *zerolog.Logger
	Metrics *observability.Metrics
}

type WorkflowEvent struct {
	WorkflowID string `json:"workflow_id"`
	Status     string `json:"status"`
	Timestamp  string `json:"timestamp"`
}

func (a *ActivityContext) ValidateAccount(ctx context.Context, req models.TransferRequest) error {
	var fromAccount, toAccount models.BankAccount
	if err := a.DB.Where("id = ?", req.FromAccountID).First(&fromAccount).Error; err != nil {
		a.Metrics.ActivityFailures.WithLabelValues("ValidateAccount").Inc()
		a.Logger.Error().Err(err).Msgf("Invalid source account: %s", req.FromAccountID)
		return fmt.Errorf("invalid source account: %w", err)
	}
	if err := a.DB.Where("id = ?", req.ToAccountID).First(&toAccount).Error; err != nil {
		a.Metrics.ActivityFailures.WithLabelValues("ValidateAccount").Inc()
		a.Logger.Error().Err(err).Msgf("Invalid destination account: %s", req.ToAccountID)
		return fmt.Errorf("invalid destination account: %w", err)
	}
	if fromAccount.Balance < req.Amount {
		a.Metrics.ActivityFailures.WithLabelValues("ValidateAccount").Inc()
		a.Logger.Error().Msgf("Insufficient funds in account %s: balance=%.2f, amount=%.2f", req.FromAccountID, fromAccount.Balance, req.Amount)
		return fmt.Errorf("insufficient funds in account %s", req.FromAccountID)
	}
	return nil
}

func (a *ActivityContext) WithdrawFunds(ctx context.Context, req models.TransferRequest) error {
	tx := a.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			a.Metrics.ActivityFailures.WithLabelValues("WithdrawFunds").Inc()
			a.Logger.Error().Interface("panic", r).Msg("Panic in WithdrawFunds")
		}
	}()
	if err := tx.Model(&models.BankAccount{}).
		Where("id = ? AND balance >= ?", req.FromAccountID, req.Amount).
		Update("balance", gorm.Expr("balance - ?", req.Amount)).Error; err != nil {
		tx.Rollback()
		a.Metrics.ActivityFailures.WithLabelValues("WithdrawFunds").Inc()
		a.Logger.Error().Err(err).Msgf("Withdraw failed for account %s", req.FromAccountID)
		return fmt.Errorf("withdraw failed: %w", err)
	}
	return tx.Commit().Error
}

func (a *ActivityContext) DepositFunds(ctx context.Context, req models.TransferRequest) error {
	tx := a.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			a.Metrics.ActivityFailures.WithLabelValues("DepositFunds").Inc()
			a.Logger.Error().Interface("panic", r).Msg("Panic in DepositFunds")
		}
	}()
	if err := tx.Model(&models.BankAccount{}).
		Where("id = ?", req.ToAccountID).
		Update("balance", gorm.Expr("balance + ?", req.Amount)).Error; err != nil {
		tx.Rollback()
		a.Metrics.ActivityFailures.WithLabelValues("DepositFunds").Inc()
		a.Logger.Error().Err(err).Msgf("Deposit failed for account %s", req.ToAccountID)
		return fmt.Errorf("deposit failed: %w", err)
	}
	return tx.Commit().Error
}

func (a *ActivityContext) LogTransaction(ctx context.Context, req models.TransferRequest) (string, error) {
	txID := uuid.New().String()
	logEntry := models.TransactionLog{
		ID:            txID,
		FromAccountID: req.FromAccountID,
		ToAccountID:   req.ToAccountID,
		Amount:        req.Amount,
	}
	if err := a.DB.Create(&logEntry).Error; err != nil {
		a.Metrics.ActivityFailures.WithLabelValues("LogTransaction").Inc()
		a.Logger.Error().Err(err).Msg("Failed to log transaction")
		return "", fmt.Errorf("failed to log transaction: %w", err)
	}
	return txID, nil
}

func (a *ActivityContext) PublishEvent(ctx context.Context, workflowID, status string) error {
	event := WorkflowEvent{
		WorkflowID: workflowID,
		Status:     status,
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
	}
	data, err := json.Marshal(event)
	if err != nil {
		a.Logger.Error().Err(err).Msg("Failed to marshal event")
		return fmt.Errorf("failed to marshal event: %w", err)
	}
	if err := a.NATS.Publish("meshery.events", data); err != nil {
		a.Logger.Error().Err(err).Msg("Failed to publish event")
		return fmt.Errorf("failed to publish event: %w", err)
	}
	return nil
}
