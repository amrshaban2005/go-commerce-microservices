package handler

import (
	"context"
	"errors"
	"net/http"

	"github.com/amrshaban2005/go-commerce-microservices/api-gateway/internal/dto"
	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func writeServiceError(c *gin.Context, err error, defaultStatus int, message string) {
	if errors.Is(err, context.DeadlineExceeded) || status.Code(err) == codes.DeadlineExceeded {
		c.JSON(http.StatusGatewayTimeout, dto.ErrorResponse{Error: "request timed out"})
		return
	}

	httpStatus := defaultStatus
	switch status.Code(err) {
	case codes.InvalidArgument:
		httpStatus = http.StatusBadRequest
	case codes.NotFound:
		httpStatus = http.StatusNotFound
	case codes.Unavailable:
		httpStatus = http.StatusBadGateway
	}

	c.JSON(httpStatus, dto.ErrorResponse{Error: message + err.Error()})
}
