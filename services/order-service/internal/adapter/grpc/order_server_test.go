package grpcadapter

import (
	"context"
	"errors"
	"testing"

	"github.com/amrshaban2005/go-commerce-microservices/services/order-service/internal/port"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestServiceErrorMapsContextErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want codes.Code
	}{
		{name: "deadline", err: context.DeadlineExceeded, want: codes.DeadlineExceeded},
		{name: "wrapped deadline", err: errors.Join(errors.New("database query"), context.DeadlineExceeded), want: codes.DeadlineExceeded},
		{name: "canceled", err: context.Canceled, want: codes.Canceled},
		{name: "not found", err: port.ErrOrderNotFound, want: codes.NotFound},
		{name: "fallback", err: errors.New("database unavailable"), want: codes.Internal},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := serviceError(test.err, codes.Internal)
			if got := status.Code(err); got != test.want {
				t.Fatalf("expected code %s, got %s", test.want, got)
			}
		})
	}
}
