package api

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/Namanv0509/BankTransfer-Temporal-Workflow/internal/models"
	"go.temporal.io/sdk/client"
)

func StartTransfer(c *gin.Context, tc client.Client, logger *zerolog.Logger) {
	var req models.TransferRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error().Err(err).Msg("Invalid request")
		c.JSON(400, gin.H{"error": "Invalid request"})
		return
	}
	workflowOptions := client.StartWorkflowOptions{
		ID:        uuid.New().String(),
		TaskQueue: "meshery-transfer-queue",
	}
	we, err := tc.ExecuteWorkflow(context.Background(), workflowOptions, "InterbankTransferWorkflow", req)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to start workflow")
		c.JSON(500, gin.H{"error": "Failed to start workflow"})
		return
	}
	c.JSON(200, gin.H{"workflow_id": we.GetID(), "status": "Started"})
}

func GetStatus(c *gin.Context, tc client.Client) {
	workflowID := c.Param("workflowID")
	workflow, err := tc.DescribeWorkflowExecution(context.Background(), workflowID, "")
	if err != nil {
		c.JSON(404, gin.H{"error": "Workflow not found"})
		return
	}
	c.JSON(200, gin.H{"workflow_id": workflowID, "status": workflow.GetWorkflowExecutionInfo().Status.String()})
}
