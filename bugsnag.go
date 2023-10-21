package kirkbugsnag

import (
	"context"
	"fmt"
	"strconv"

	"github.com/bugsnag/bugsnag-go/v2"
	"github.com/pkg/errors"

	"github.com/jimenezmaximiliano/kirk"
)

// Configuration is the configuration for the bugsnag reporter.
type Configuration struct {
	ApiKey      string
	Environment string
}

// SetupDefaultBugsnag configures the bugsnag reporter with the given configuration.
// This works globally, unfortunately.
func SetupDefaultBugsnag(config Configuration) {
	bugsnag.Configure(bugsnag.Configuration{
		APIKey:       config.ApiKey,
		ReleaseStage: config.Environment,
		PanicHandler: func() {}, // We don't want to automatically report panics.
	})

	bugsnag.OnBeforeNotify(
		func(event *bugsnag.Event, config *bugsnag.Configuration) error {
			// Set the error message.
			event.ErrorClass = event.Error.Error()

			// Set the stacktrace.
			type stackTracer interface {
				StackTrace() errors.StackTrace
			}

			if err, ok := event.Error.Err.(stackTracer); ok {
				var stacktrace []bugsnag.StackFrame
				for _, item := range err.StackTrace() {
					lineNumber, err := strconv.Atoi(fmt.Sprintf("%d", item))
					if err != nil {
						// We'll just skip this stacktrace item.
						continue
					}

					stacktrace = append(stacktrace, bugsnag.StackFrame{
						Method:     fmt.Sprintf("%n", item),
						File:       fmt.Sprintf("%s", item),
						LineNumber: lineNumber,
						InProject:  true,
					})
				}

				event.Stacktrace = stacktrace
			}

			return nil
		})
}

// ReporterAdapter is an adapter for the bugsnag reporter.
type ReporterAdapter struct {
	logger kirk.LoggerForReporter
}

// Make sure ReporterAdapter implements kirk.Reporter.
var _ kirk.Reporter = ReporterAdapter{}

// NewReporterAdapter creates a new bugsnag reporter adapter.
func NewReporterAdapter(logger kirk.LoggerForReporter) ReporterAdapter {
	return ReporterAdapter{
		logger: logger,
	}
}

// ReportError reports the given error to bugsnag.
func (reporter ReporterAdapter) ReportError(ctx context.Context, errorToReport error) {
	fields := kirk.FieldsFromCtx(ctx)
	metadata := make(map[string]interface{}, len(fields))

	for k, v := range fields {
		metadata[k] = v
	}

	if len(fields) > 0 {
		err := bugsnag.Notify(errorToReport, bugsnag.MetaData{
			"fields": metadata,
		})
		if err != nil {
			reporter.logger.Error(ctx, errors.Wrap(err, "failed to report error to bugsnag"))
		}

		return
	}

	if err := bugsnag.Notify(errorToReport); err != nil {
		reporter.logger.Error(ctx, errors.Wrap(err, "failed to report error to bugsnag"))
	}
}
