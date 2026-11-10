package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"github.com/Nischaldh/linko/internal/store"
	pkgerr "github.com/pkg/errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type closeFunc func() error

type stackTracer interface {
	error
	StackTrace() pkgerr.StackTrace
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)

	httpPort := flag.Int("port", 8899, "port to listen on")
	dataDir := flag.String("data", "./data", "directory to store data")
	flag.Parse()
	status := run(ctx, cancel, *httpPort, *dataDir)
	cancel()
	os.Exit(status)
}

func run(ctx context.Context, cancel context.CancelFunc, httpPort int, dataDir string) int {
	logger, close, err := initializeLogger()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize logger: %v\n", err)
		return 1
	}
	defer func() {
		if err := close(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to close logger: %v\n", err)
		}
	}()
	st, err := store.New(dataDir, logger)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to create store: %v\n", err))
		return 1
	}
	s := newServer(*st, httpPort, cancel, logger)
	var serverErr error
	go func() {
		serverErr = s.start()
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	logger.Debug("Linko is shutting down")
	if err := s.shutdown(shutdownCtx); err != nil {
		logger.Error(fmt.Sprintf("failed to shutdown server: %v", err))
		return 1
	}
	if serverErr != nil {
		logger.Error(fmt.Sprintf("server error: %v", serverErr))
		return 1
	}
	return 0
}

func initializeLogger() (*slog.Logger, closeFunc, error) {
	handlers := []slog.Handler{
		slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			ReplaceAttr: replaceAttr,
			Level:       slog.LevelDebug,
		}),
	}
	closers := []closeFunc{}
	logFile, exists := os.LookupEnv("LINKO_LOG_FILE")
	if exists {
		file, err := os.OpenFile(logFile, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o644)
		if err != nil {
			return nil, nil, err
		}

		bufferedFile := bufio.NewWriterSize(file, 8192)
		close := func() error {
			err := bufferedFile.Flush()
			if err != nil {
				return err
			}
			if err = file.Close(); err != nil {
				return err
			}
			return nil
		}
		handlers = append(handlers, slog.NewJSONHandler(bufferedFile, &slog.HandlerOptions{
			ReplaceAttr: replaceAttr,
			Level:       slog.LevelInfo,
		}))
		closers = append(closers, close)
	}
	closer := func() error {
		var errs []error
		for _, close := range closers {
			if err := close(); err != nil {
				errs = append(errs, err)
			}
		}
		return errors.Join(errs...)
	}
	return slog.New(slog.NewMultiHandler(handlers...)), closer, nil

}

func replaceAttr(groups []string, a slog.Attr) slog.Attr {
	if a.Key == "error" {
		err, ok := a.Value.Any().(error)
		if !ok {
			return a
		}
		if stackErr, ok := errors.AsType[stackTracer](err); ok {
			return slog.GroupAttrs("error",
				slog.Attr{
					Key:   "message",
					Value: slog.StringValue(stackErr.Error()),
				},
				slog.Attr{
					Key: "stack_trace",
					Value: slog.StringValue(
						fmt.Sprintf("%+v", stackErr.StackTrace()),
					),
				},
			)
		}

		return slog.String("error", fmt.Sprintf("%+v", err))
	}
	return a
}
