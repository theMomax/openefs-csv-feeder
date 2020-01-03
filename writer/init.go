package writer

import (
	"github.com/sirupsen/logrus"
	"github.com/theMomax/openefs-csv-feeder/config"
)

// Config paths
const (
	PathAddress = "writer.address"
)

func init() {
	config.RootCtx.PersistentFlags().StringP(PathAddress, "a", "http://localhost:8080", "openefs server address")
	config.Viper.BindPFlag(PathAddress, config.RootCtx.PersistentFlags().Lookup(PathAddress))
	config.OnInitialize(func() {
		log = config.NewLogger()
	})
}

var log *logrus.Logger

type Writer struct {
	prodaddress    string
	weatheraddress string
}

func NewWriter(address string) *Writer {
	return &Writer{
		prodaddress:    address + "/v1/input/production/:unixtimestamp/",
		weatheraddress: address + "/v1/input/weather/:unixtimestamp/",
	}
}

func NewWriterFromConfig() *Writer {
	return NewWriter(config.Viper.GetString(PathAddress))
}
