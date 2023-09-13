package main

import (
	"context"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/emvi/logbuch"
	ghandlers "github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	conf "github.com/muety/mailwhale/config"
	"github.com/muety/mailwhale/service"
	"github.com/muety/mailwhale/web/handlers"
	"github.com/muety/mailwhale/web/routes/api"
	"github.com/rs/cors"
	"github.com/timshannon/bolthold"
)

var (
	config      *conf.Config
	store       *bolthold.Store
	userService *service.UserService
)

func main() {
	config = conf.Load()
	store = conf.LoadStore(config.Store.Path)
	defer store.Close()

	// Set log level
	if config.IsDev() {
		logbuch.SetLevel(logbuch.LevelDebug)
	} else {
		logbuch.SetLevel(logbuch.LevelInfo)
	}

	// Services
	userService = service.NewUserService()

	// Global middlewares
	recoverMiddleware := ghandlers.RecoveryHandler()
	loggingMiddleware := handlers.NewLoggingMiddleware(logbuch.Info, []string{})

	// CORS
	corsHandler := cors.New(cors.Options{
		AllowedOrigins: config.Web.CorsOrigins,
		AllowedMethods: []string{
			http.MethodHead,
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
		},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	})

	// Configure routing
	router := mux.NewRouter().StrictSlash(true)
	router.Use(recoverMiddleware, loggingMiddleware)

	// Handlers
	api.NewHealthHandler().Register(router)
	api.NewMailHandler().Register(router)
	api.NewClientHandler().Register(router)
	api.NewUserHandler().Register(router)
	api.NewTemplateHandler().Register(router)

	handler := corsHandler.Handler(router)

	// Static routes
	router.PathPrefix("/").Handler(&handlers.SPAHandler{
		StaticPath:      "./webui/public",
		IndexPath:       "index.html",
		ReplaceBasePath: config.Web.GetPublicUrl() + "/",
		NoCache:         config.IsDev(),
	})

	listen(handler, config)

}

func listen(handler http.Handler, config *conf.Config) {
	var s4 *http.Server
	ctx, cancel := context.WithCancel(context.Background())

	s4 = &http.Server{
		Handler:      handler,
		Addr:         config.Web.ListenAddr,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		BaseContext:  func(_ net.Listener) context.Context { return ctx },
	}

	go func() {
		logbuch.Info("web server started, listening on %s", config.Web.ListenAddr)
		if err := s4.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logbuch.Fatal("failed to start web server: %v", err)
		}
	}()

	signalChan := make(chan os.Signal, 1)

	signal.Notify(
		signalChan,
		syscall.SIGHUP,  // kill -SIGHUP XXXX
		syscall.SIGINT,  // kill -SIGINT XXXX or Ctrl+c
		syscall.SIGQUIT, // kill -SIGQUIT XXXX
	)

	<-signalChan
	logbuch.Info("os.Interrupt - shutting down...\n")

	go func() {
		<-signalChan
		logbuch.Fatal("os.Kill - terminating...\n")
	}()

	gracefullCtx, cancelShutdown := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelShutdown()

	if err := s4.Shutdown(gracefullCtx); err != nil {
		logbuch.Warn("shutdown error: %v\n", err)
		defer os.Exit(1)
		return
	} else {
		logbuch.Info("gracefully stopped\n")
	}

	// manually cancel context if not using server.RegisterOnShutdown(cancel)
	cancel()

	defer os.Exit(0)
	return
}
