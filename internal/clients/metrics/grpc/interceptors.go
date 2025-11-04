package grpc

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/kdv2001/onlyMetrics/internal/domain"
)

// NewErrorInterceptor создает Interceptor для приведения транспортной ошибки к доменной.
func NewErrorInterceptor() grpc.UnaryClientInterceptor {
	return func(ctx context.Context,
		method string,
		req, reply any,
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption) error {
		err := invoker(ctx, method, req, reply, cc, opts...)
		if err != nil {
			s := status.Convert(err)
			switch s.Code() {
			case codes.InvalidArgument:
				return fmt.Errorf("%s: %w", s.Message(), domain.ErrInvalidArgument)
			case codes.DeadlineExceeded:
				return fmt.Errorf("%s: %w", s.Message(), context.DeadlineExceeded)
			case codes.Canceled:
				return fmt.Errorf("%s: %w", s.Message(), context.Canceled)
			case codes.NotFound:
				return fmt.Errorf("%s: %w", s.Message(), domain.ErrNotFound)
			case codes.FailedPrecondition:
				return fmt.Errorf("%s: %w", s.Message(), domain.ErrResourceIsLocked)
			}

			return fmt.Errorf("internal server error: %s", s.Message())
		}

		return nil
	}

}
