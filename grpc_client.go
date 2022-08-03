package u

import (
	"time"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_zap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/keepalive"
)

//type GRPCConnection struct {
//	Conn *grpc.ClientConn
//	host string
//}
//
//func NewGRPCConnection(host string) GRPCConnection {
//	return GRPCConnection{
//		host: host,
//	}
//}

func DialGRPC(host string, panicIfErrorOccurred bool) (*grpc.ClientConn, ErrorType) {
	// Set up a connection to the server.
	var err error
	backoffCfg := backoff.DefaultConfig
	backoffCfg.MaxDelay = 3 * time.Second // 最多间隔MaxDelay秒重新尝试连接

	// Discussion
	// With grpc.WithBlock() option set, grpc.Dial() will be blocked until connection be made.
	// Without grpc.WithBlock() option set, if connection cannot be made yet, Dial() returns a ClientConn object and no error anyway.
	// It seems Connection Backoff will handle retry connecting.
	conn, err := grpc.Dial(
		host,
		grpc.WithInsecure(),
		//grpc.WithBlock(),
		grpc.WithConnectParams(grpc.ConnectParams{
			MinConnectTimeout: 10, // 如果建立连接需要10秒，服务端或网络有问题。
			Backoff:           backoff.DefaultConfig,
		}),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                10 * time.Second,
			Timeout:             10 * time.Second,
			PermitWithoutStream: true,
		}),
		grpc.WithChainUnaryInterceptor(grpc_middleware.ChainUnaryClient(
			grpc_zap.UnaryClientInterceptor(Logger, GRPCClientZapLogOption()),
			grpc_prometheus.UnaryClientInterceptor,
		)),
		grpc.WithChainStreamInterceptor(grpc_middleware.ChainStreamClient(
			grpc_zap.StreamClientInterceptor(Logger, GRPCClientZapLogOption()),
			grpc_prometheus.StreamClientInterceptor,
		)),
	)

	if err != nil {
		erro := ErrGRPCDialErr(host, err)
		if panicIfErrorOccurred {
			panic(erro)
		}
		//Error("Can't dial to grpc server %v. error=%v", c.host, err)
		return nil, erro
	}
	Infof("Create connection to GRPC Server %s", host)

	return conn, nil
}
