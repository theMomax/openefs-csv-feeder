package config

import (
	"github.com/sirupsen/logrus"
)

// Log formatter options
const (
	json   = "json"
	logfmt = "logfmt"
	tty    = "tty"
)

// Config paths
const (
	PathLevel     = "loglevel"
	PathFormatter = "logformatter"
)

func init() {
	// provide configuration
	RootCtx.PersistentFlags().UintP(PathLevel, "l", uint(logrus.InfoLevel), "log level (Panic: 0, Fatal: 1, Error: 2, Warning: 3, Info: 4, Debug: 5, Trace: 6)")
	Viper.BindPFlag(PathLevel, RootCtx.PersistentFlags().Lookup(PathLevel))

	RootCtx.PersistentFlags().String(PathFormatter, tty, "log format")
	Viper.BindPFlag(PathFormatter, RootCtx.PersistentFlags().Lookup(PathFormatter))
}

func initializeLogrus() {
	// check logging level
	lvl := Viper.GetUint32(PathLevel)
	if uint32(logrus.TraceLevel) < lvl {
		InvalidConfiguration(PathLevel, [...]logrus.Level{logrus.PanicLevel, logrus.FatalLevel, logrus.ErrorLevel, logrus.WarnLevel, logrus.InfoLevel, logrus.DebugLevel, logrus.TraceLevel})
	}
	// check logging format
	LogFormatter()
}

// NewLogger returns a new logger instance as configured by this package's
// viper instance.
func NewLogger() *logrus.Logger {
	l := logrus.New()
	// choose formatter
	l.SetFormatter(LogFormatter())

	l.SetLevel(logrus.Level(Viper.GetUint32(PathLevel)))
	return l
}

// LogFormatter returns the configured logrus formatter.
func LogFormatter() logrus.Formatter {
	switch Viper.GetString(PathFormatter) {
	case json:
		return &logrus.JSONFormatter{}
	case logfmt:
		return &logrus.TextFormatter{
			DisableColors: true,
			FullTimestamp: true,
		}
	case tty:
		return &logrus.TextFormatter{}
	default:
		InvalidConfiguration(PathFormatter, [...]string{json, logfmt, tty})
		return nil
	}
}
