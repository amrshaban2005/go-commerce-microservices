package messaging

import (
	"testing"

	amqp "github.com/rabbitmq/amqp091-go"
)

func TestDeliveryAttempt(t *testing.T) {
	tests := []struct {
		name    string
		headers amqp.Table
		want    int
	}{
		{name: "first delivery", want: 1},
		{name: "int32 header", headers: amqp.Table{deliveryAttemptHeader: int32(2)}, want: 2},
		{name: "int64 header", headers: amqp.Table{deliveryAttemptHeader: int64(3)}, want: 3},
		{name: "invalid header", headers: amqp.Table{deliveryAttemptHeader: "three"}, want: 1},
		{name: "non-positive header", headers: amqp.Table{deliveryAttemptHeader: int32(0)}, want: 1},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := deliveryAttempt(test.headers); got != test.want {
				t.Fatalf("deliveryAttempt() = %d, want %d", got, test.want)
			}
		})
	}
}

func TestCopyHeadersDoesNotMutateOriginal(t *testing.T) {
	original := amqp.Table{"existing": "value"}
	copied := copyHeaders(original)
	copied[deliveryAttemptHeader] = int32(2)

	if _, exists := original[deliveryAttemptHeader]; exists {
		t.Fatal("copyHeaders mutated the original headers")
	}
}
