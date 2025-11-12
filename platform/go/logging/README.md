# Logging Helpers

The `platform/go/logging` package wraps `zap` to produce Google Cloud Logging–compatible JSON and request-scoped helpers you can reuse across services.

## Features

- `NewLogger` builds a base `zap.Logger` with `severity`, `message`, and `timestamp` fields that map cleanly to Cloud Logging.
- `RequestLogger` is an HTTP middleware that enriches logs with `request_id`, path, method, remote address, status code, bytes written, and request duration.
- `WithLogger` / `FromContext` let downstream code pull a request-specific logger from the `context.Context`.

## Usage

```go
import (
    "net/http"
    "github.com/go-chi/chi/v5"

    platformlogging "github.com/TCGLandDev/tcgdb/platform/go/logging"
)

func main() {
    logger, err := platformlogging.NewLogger(platformlogging.Config{
        Component: "api-server",
        Level:     "debug",
    })
    if err != nil {
        panic(err)
    }
    defer logger.Sync()

    router := chi.NewRouter()
    router.Use(platformlogging.RequestLogger(logger))

    router.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
        platformlogging.FromRequest(r, logger).Info("handling ping")
        w.WriteHeader(http.StatusOK)
        _, _ = w.Write([]byte("pong"))
    })

    if err := http.ListenAndServe(":3000", router); err != nil {
        logger.Fatal("server failed", zap.Error(err))
    }
}
```

The middleware automatically annotates each log entry with the request information, and nested handlers can call `FromContext` to add structured fields or additional events during request processing.

### Non-HTTP usage

The logger factory can be used outside HTTP servers—just call `NewLogger` once and reuse it across, for example, goroutines:

```go
func runWorker(ctx context.Context) {
    logger, err := platformlogging.NewLogger(platformlogging.Config{
        Component: "worker",
        Level:     "info",
    })
    if err != nil {
        panic(err)
    }
    defer logger.Sync()

    logger.Info("starting background job")
    // ... work ...
    if err := doWork(ctx); err != nil {
        logger.Error("job failed", zap.Error(err))
        return
    }
    logger.Info("job complete")
}
```
