package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/joho/godotenv"

	"blog-api/internal/handler"
	"blog-api/internal/middleware"
	"blog-api/internal/model"
	"blog-api/internal/repository"
	"blog-api/internal/service"
	"blog-api/pkg/auth"
	"blog-api/pkg/database"
	"blog-api/pkg/logging"
	"blog-api/pkg/settings"
	"blog-api/pkg/throttle"
)

var logger = logging.L()

const gracefulShutDownTimeout = 30 * time.Second

func main() {

	// config
	if err := godotenv.Load(); err != nil {
		logger.Warn(".env file not found, using environment variables")
	}
	dbConfig := &database.DatabaseConfig{}
	jwtConfig := &auth.JWTConfig{}
	passConfig := &auth.PasswordConfig{}
	redisConfig := &middleware.RedisConfig{}
	loggingConfig := &logging.LoggerConfig{}
	for _, cfg := range []settings.EnvConfigurable{
		dbConfig,
		jwtConfig,
		passConfig,
		redisConfig,
		loggingConfig,
	} {
		settings.LoadConfig(cfg)
	}

	// logger
	logging.Init(loggingConfig)
	logger.Info("logger initialized")

	// database
	db, err := database.NewDatabaseManager(dbConfig)
	if err != nil {
		panic(err)
	}
	defer db.Dispose()
	if err := db.InitORM(); err != nil {
		panic(fmt.Errorf("failed to initialize ORM: %w", err))
	}

	// auth
	jwtManager := auth.NewJWTManager(jwtConfig)
	passManager := auth.NewPasswordManager(passConfig)

	// repos
	userRepo := repository.NewUserRepo(db)
	refreshTokenRepo := repository.NewRefreshTokenRepo(db)
	postRepo := repository.NewPostRepo(db)
	commentRepo := repository.NewCommentRepo(db)

	// services
	userService := service.NewUserService(userRepo, refreshTokenRepo, jwtManager, passManager)
	postService := service.NewPostService(postRepo, userRepo)
	commentService := service.NewCommentService(commentRepo, postRepo, userRepo)

	// post scheduler
	schedulerCtx, schedulerCancel := context.WithCancel(context.Background())
	go service.StartPostScheduler(schedulerCtx, postRepo)

	// handlers
	userHandler := handler.NewAuthHandler(userService)
	postHandler := handler.NewPostHandler(postService)
	commentHandler := handler.NewCommentHandler(commentService)

	authMiddleware := middleware.NewAuthMiddleware(jwtManager)

	router := chi.NewRouter()

	// global middlewares
	globalMiddleware := []func(http.Handler) http.Handler{
		middleware.PanicRecoverMiddleware,
		middleware.CORSMiddleware,
		middleware.XRayMiddleware,
		middleware.RequestLoggerMiddleware,
	}

	// throttling
	throttle.InitRedis(fmt.Sprintf("%s:%d", redisConfig.Host, redisConfig.Port), redisConfig.Password, redisConfig.DB)

	chain := middleware.Chain(router, globalMiddleware...)

	// health check
	router.Get("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok","service":"blog-api"}`))
	})

	// auth
	authThrottler := throttle.NewThrottler("auth", 10, time.Minute)

	router.With(
		middleware.RateLimiterMiddleware(authThrottler),
	).Post(
		"/api/register",
		middleware.ModelBodyMiddleware[model.UserCreateRequest](userHandler.Register),
	)

	router.With(
		middleware.RateLimiterMiddleware(authThrottler),
	).Post(
		"/api/login",
		middleware.ModelBodyMiddleware[model.UserLoginRequest](userHandler.Login),
	)

	router.With(
		middleware.RateLimiterMiddleware(authThrottler),
	).Post(
		"/api/refresh",
		middleware.ModelBodyMiddleware[model.RefreshTokenRequest](userHandler.Refresh),
	)

	// public post endpoints
	router.Get("/api/posts", postHandler.GetAll)
	router.Get("/api/posts/{postID}", postHandler.GetByID)
	router.Get("/api/posts/{postID}/comments", commentHandler.GetByPost)

	// protected routes
	protected := chi.NewRouter()
	protected.Use(authMiddleware.RequireAuth)

	// body requests
	protected.Post(
		"/api/posts",
		middleware.ModelBodyMiddleware[model.PostCreateRequest](postHandler.Create),
	)
	protected.Put(
		"/api/posts/{postID}",
		middleware.ModelBodyMiddleware[model.PostUpdateRequest](postHandler.Update),
	)
	protected.Delete("/api/posts/{postID}", postHandler.Delete)

	protected.Post(
		"/api/posts/{postID}/comments",
		middleware.ModelBodyMiddleware[model.CommentCreateRequest](commentHandler.Create),
	)
	protected.Put(
		"/api/posts/{postID}/comments/{commentID}",
		middleware.ModelBodyMiddleware[model.CommentUpdateRequest](commentHandler.Update),
	)
	protected.Delete("/api/posts/{postID}/comments/{commentID}", commentHandler.Delete)

	// me
	protected.Get("/api/users/{userID}", userHandler.GetProfile)

	// delayed posts
	protected.Get("/api/delayed", postHandler.GetAllDelayed)
	protected.Get("/api/delayed/{postID}", postHandler.GetDelayedByID)

	router.Mount("/", protected)
	host := os.Getenv("HOST")
	if host == "" {
		host = "0.0.0.0"
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	addr := host + ":" + port

	srv := &http.Server{
		Addr:         addr,
		Handler:      chain,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		logger.Info("starting server at %s...", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Server error: %v", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	<-quit
	logger.Info("shutdown signal received")
	schedulerCancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), gracefulShutDownTimeout)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("server shutdown failed: %v", err)
	} else {
		logger.Info("server shut down gracefully")
	}
}
