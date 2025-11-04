package grpc

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/kdv2001/onlyMetrics/internal/domain"
	"github.com/kdv2001/onlyMetrics/pkg/logger"
	"github.com/kdv2001/onlyMetrics/pkg/network"
)

// AddLoggerToContextInterceptor Interceptor для помещения logger в context.
func AddLoggerToContextInterceptor(sugarLogger *zap.SugaredLogger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		ctx = logger.ToContext(ctx, sugarLogger)
		return handler(ctx, req)
	}
}

// RequestInterceptor Interceptor для логирования запросов.
func RequestInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()
		defer func() {
			logger.Infof(ctx, "request: method: %s; processing time: %s",
				info.FullMethod, time.Since(start).String())
		}()

		return handler(ctx, req)
	}
}

// ResponseInterceptor Interceptor для логирования ответов.
func ResponseInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		req, err := handler(ctx, req)
		defer func() {
			defer func() {
				errStatus := status.Code(err)
				logger.Infof(ctx, "response: status code: %d",
					errStatus)
			}()
		}()
		return req, err
	}
}

// NewSubNetInterceptor создает Interceptor для проверки принадлежности запрос клиента к подсети
func NewSubNetInterceptor(cidr string) (grpc.UnaryServerInterceptor, error) {
	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, fmt.Errorf("error parse CIdr %w", err)
	}

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.PermissionDenied, "forbidden")
		}

		val := md.Get(network.XRealIP)
		if len(val) == 0 {
			return nil, status.Error(codes.PermissionDenied, "forbidden")
		}

		inputIP := net.ParseIP(val[0])
		if inputIP == nil {
			return nil, status.Error(codes.PermissionDenied, "invalid ip")
		}

		if !ipNet.Contains(inputIP) {
			return nil, status.Error(codes.PermissionDenied, "forbidden")
		}

		return handler(ctx, req)

	}, nil
}

// NewErrorInterceptor создает Interceptor для приведения доменной ошибки к транспортной.
func NewErrorInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		resp, err := handler(ctx, req)
		if err != nil {
			switch {
			case errors.Is(err, context.DeadlineExceeded):
				return nil, status.Error(codes.DeadlineExceeded, err.Error())
			case errors.Is(err, context.Canceled):
				return nil, status.Error(codes.Canceled, err.Error())
			case errors.Is(err, domain.ErrNotFound):
				return nil, status.Error(codes.NotFound, err.Error())
			case errors.Is(err, domain.ErrResourceIsLocked):
				return nil, status.Error(codes.FailedPrecondition, err.Error())
			case errors.Is(err, domain.ErrInvalidArgument):
				return nil, status.Error(codes.InvalidArgument, err.Error())
			}

			return nil, status.Error(codes.Internal, "internal server error")
		}

		return resp, nil
	}
}
