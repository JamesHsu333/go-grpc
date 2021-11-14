package server

import (
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-redis/redis/v8"
	grpcrecovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	grpc_ctxtags "github.com/grpc-ecosystem/go-grpc-middleware/tags"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/jmoiron/sqlx"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"

	"github.com/JamesHsu333/go-grpc/config"
	"github.com/JamesHsu333/go-grpc/internal/interceptors"
	sessRepository "github.com/JamesHsu333/go-grpc/internal/session/repository"
	sessUseCase "github.com/JamesHsu333/go-grpc/internal/session/usecase"
	userServerGRPC "github.com/JamesHsu333/go-grpc/internal/user/delivery/grpc"
	userRepository "github.com/JamesHsu333/go-grpc/internal/user/repository"
	userUseCase "github.com/JamesHsu333/go-grpc/internal/user/usecase"
	"github.com/JamesHsu333/go-grpc/pkg/logger"
	"github.com/JamesHsu333/go-grpc/pkg/metric"
	userProto "github.com/JamesHsu333/go-grpc/proto/user"
)

// GRPC Server
type Server struct {
	cfg         *config.Config
	db          *sqlx.DB
	redisClient *redis.Client
	logger      logger.Logger
}

// NewServer New Server constructor
func NewServer(cfg *config.Config, db *sqlx.DB, redisClient *redis.Client, logger logger.Logger) *Server {
	return &Server{cfg: cfg, db: db, redisClient: redisClient, logger: logger}
}

func (s *Server) Run() error {
	metrics, err := metric.CreateMetrics(s.cfg.Metrics.URL, s.cfg.Metrics.ServiceName)
	if err != nil {
		s.logger.Errorf("CreateMetrics Error: %s", err)
	}
	s.logger.Infof(
		"Metrics available URL: %s, ServiceName: %s",
		s.cfg.Metrics.URL,
		s.cfg.Metrics.ServiceName,
	)

	im := interceptors.NewInterceptorManager(s.logger, s.cfg, metrics)
	userRepo := userRepository.NewUserRepository(s.db)
	sessRepo := sessRepository.NewSessionRepository(s.redisClient, s.cfg)
	userRedisRepo := userRepository.NewUserRedisRepo(s.redisClient, s.logger)
	userUC := userUseCase.NewUserUC(userRepo, userRedisRepo, s.logger)
	sessUC := sessUseCase.NewSessionUseCase(sessRepo, s.cfg)

	l, err := net.Listen("tcp", s.cfg.Server.Port)
	if err != nil {
		return err
	}
	defer l.Close()

	server := grpc.NewServer(grpc.KeepaliveParams(keepalive.ServerParameters{
		MaxConnectionIdle: s.cfg.Server.MaxConnectionIdle * time.Minute,
		Timeout:           s.cfg.Server.Timeout * time.Second,
		MaxConnectionAge:  s.cfg.Server.MaxConnectionAge * time.Minute,
		Time:              s.cfg.Server.Timeout * time.Minute,
	}),
		grpc.UnaryInterceptor(im.Logger),
		grpc.ChainUnaryInterceptor(
			grpc_ctxtags.UnaryServerInterceptor(),
			grpc_prometheus.UnaryServerInterceptor,
			grpcrecovery.UnaryServerInterceptor(),
		),
	)

	if s.cfg.Server.Mode != "Production" {
		reflection.Register(server)
	}

	userGRPCServer := userServerGRPC.NewUserServerGRPC(s.logger, s.cfg, userUC, sessUC)
	userProto.RegisterUserServiceServer(server, userGRPCServer)

	grpc_prometheus.Register(server)
	http.Handle("/metrics", promhttp.Handler())

	go func() {
		s.logger.Infof("Server is listening on port: %v", s.cfg.Server.Port)
		if err := server.Serve(l); err != nil {
			s.logger.Fatal(err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	<-quit
	server.GracefulStop()
	s.logger.Info("Server Exited Properly")

	return nil
}
