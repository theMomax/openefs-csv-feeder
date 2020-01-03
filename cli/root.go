package cli

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/theMomax/openefs-csv-feeder/mocktime"

	"github.com/spf13/cobra"
	"github.com/theMomax/openefs-csv-feeder/config"
	"github.com/theMomax/openefs-csv-feeder/reader"
	"github.com/theMomax/openefs-csv-feeder/writer"
	"github.com/theMomax/openefs/models/production"
	"github.com/theMomax/openefs/models/production/weather"
)

// Config paths
const (
	PathBatchSize = "cli.batchsize"
	PathStartTime = "cli.starttime"
)

func init() {
	config.RootCtx.Run = run

	config.RootCtx.PersistentFlags().UintP(PathBatchSize, "b", 24, "the number of hours to be processed per batch (execution will pause before each batch)")
	config.Viper.BindPFlag(PathBatchSize, config.RootCtx.PersistentFlags().Lookup(PathBatchSize))
	config.RootCtx.PersistentFlags().Int64P(PathStartTime, "s", math.MinInt64, "the unix time (in seconds) where the reader starts (if older than the oldest input-value, the latter is used)")
	config.Viper.BindPFlag(PathStartTime, config.RootCtx.PersistentFlags().Lookup(PathStartTime))
	config.OnInitialize(func() {
		log = config.NewLogger()
	})
}

var log *logrus.Logger

// Execute executes the root command.
func Execute() error {
	return config.RootCtx.Execute()
}

func run(cmd *cobra.Command, args []string) {
	w := writer.NewWriterFromConfig()
	r, err := reader.NewReaderFromConfig()
	if err != nil {
		log.Fatal(err)
	}

	batchSize := config.Viper.GetUint(PathBatchSize)
	skip := uint(0)
	count := uint(0)

	r.ForEach(
		func(date time.Time, production *production.Data) {
			log.WithField("date", date).Info("updated mocktime")
			err := mocktime.Update(date)
			if err != nil {
				log.Fatal(err)
			}

			if count%batchSize == 0 {
				if skip == 0 {
					skip = pause(date)
				} else {
					skip--
				}
				count = 1
			} else {
				count++
			}

			err = w.WriteProduction(date, production)
			if err != nil {
				log.Fatal(err)
			}
		},
		func(dates []time.Time, weather []*weather.Data) {
			for i := range dates {
				err := w.WriteWeather(dates[i], weather[i])
				if err != nil {
					log.Fatal(err)
				}
			}
		},
		time.Unix(config.Viper.GetInt64(PathStartTime), 0),
	)
	log.Info("completed")
}

func pause(date time.Time) uint {
	r := bufio.NewReader(os.Stdin)
	fmt.Printf("(%s | %d) > ", date.String(), date.Unix())
	text, _ := r.ReadString('\n')
	text = strings.TrimSpace(text)
	nr, err := strconv.ParseUint(text, 10, 64)
	if err != nil || nr == 0 {
		return 0
	}
	return uint(nr) - 1
}
