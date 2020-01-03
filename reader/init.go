package reader

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/theMomax/openefs/models/production"
	"github.com/theMomax/openefs/models/production/weather"
	timeutils "github.com/theMomax/openefs/utils/time"

	"github.com/gocarina/gocsv"
	"github.com/theMomax/openefs-csv-feeder/config"
)

// Config paths
const (
	PathWeatherBasePath = "reader.weatherbasepath"
	PathProductionPath  = "reader.productionpath"
	PathStepSize        = "reader.productionstepsize"
	PathStepAmount      = "reader.productionsteps"
)

func init() {
	config.RootCtx.PersistentFlags().StringP(PathWeatherBasePath, "w", ".", "the folder, where the weather-input-files are located")
	config.Viper.BindPFlag(PathWeatherBasePath, config.RootCtx.PersistentFlags().Lookup(PathWeatherBasePath))

	config.RootCtx.PersistentFlags().StringP(PathProductionPath, "p", ".", "the file containing production-input-data")
	config.Viper.BindPFlag(PathProductionPath, config.RootCtx.PersistentFlags().Lookup(PathProductionPath))

	config.RootCtx.PersistentFlags().Duration(PathStepSize, time.Hour, "the duration (in seconds) of a single time-step as required by the production-forecasting-model")
	config.Viper.BindPFlag(PathStepSize, config.RootCtx.PersistentFlags().Lookup(PathStepSize))

	config.RootCtx.PersistentFlags().Uint(PathStepAmount, 120, "the amount of steps (as defined by "+PathStepSize+") required by the production-forecasting-model")
	config.Viper.BindPFlag(PathStepAmount, config.RootCtx.PersistentFlags().Lookup(PathStepAmount))
	config.OnInitialize(func() {
		log = config.NewLogger()
	})
}

var log *logrus.Logger

type Reader struct {
	productionTimestep   time.Duration
	oldestProductionData *time.Time
	latestProductionData *time.Time
	production           map[time.Time]*production.Data
	weather              map[time.Duration]map[time.Time]*weather.Data
	oldestWeatherData    map[time.Duration]*time.Time
	latestWeatherData    map[time.Duration]*time.Time
	forecastPoints       []time.Duration
	round                func(t time.Time) time.Time
}

func NewReader(weatherBaseAddress string, productionAddress string, timestep time.Duration, stepAmount uint) (*Reader, error) {
	log.WithFields(logrus.Fields{
		"weatherBaseAddress": weatherBaseAddress,
		"productionAddress":  productionAddress,
		"timestep":           timestep,
		"stepAmount":         stepAmount,
	}).Info("creating new reader...")

	round := func(date time.Time) time.Time {
		r := timeutils.Round(date, timestep)
		if r.Unix() != date.Unix() {
			log.WithField("actual", date).WithField("rounded", r).Trace("rounded input-timestamp")
		}
		return r
	}

	wold, wlatest, forecastPoints, wd, err := readWeatherInput(weatherBaseAddress, round, stepAmount)
	if err != nil {
		return nil, err
	}

	pold, platest, pd, err := readProductionInput(productionAddress, round)
	if err != nil {
		return nil, err
	}

	return &Reader{
		production:           pd,
		oldestProductionData: pold,
		latestProductionData: platest,
		weather:              wd,
		oldestWeatherData:    wold,
		latestWeatherData:    wlatest,
		forecastPoints:       forecastPoints,
		productionTimestep:   timestep,
		round:                round,
	}, nil
}

func NewReaderFromConfig() (*Reader, error) {
	return NewReader(config.Viper.GetString(PathWeatherBasePath), config.Viper.GetString(PathProductionPath), config.Viper.GetDuration(PathStepSize), config.Viper.GetUint(PathStepAmount))
}

func readWeatherInput(weatherBasePath string, round func(time.Time) time.Time, stepAmount uint) (oldest, latest map[time.Duration]*time.Time, forecastPoints []time.Duration, data map[time.Duration]map[time.Time]*weather.Data, err error) {
	log.Info("reading weather data...")

	type fileInfo struct {
		path     string
		isHourly bool
	}
	files := make(map[time.Duration]fileInfo, 0)

	err = filepath.Walk(weatherBasePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		log.WithField("filepath", path).Trace("found candidate-file")
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".csv") {
			if !strings.HasPrefix(info.Name(), "forecast_") || !strings.HasSuffix(info.Name(), "_ahead.csv") {
				log.WithField("filepath", path).Warning("possible input-file does not match pattern 'forecast_*(h|d)_ahead.csv', where * is a non-negative integer (prefix/suffix not matching)")
				return nil
			}
			durationDescription := strings.TrimSuffix(strings.TrimPrefix(info.Name(), "forecast_"), "_ahead.csv")
			var multiplier time.Duration
			switch durationDescription[len(durationDescription)-1] {
			case 'h':
				multiplier = time.Hour
			case 'd':
				multiplier = 24 * time.Hour
			default:
				log.WithField("filepath", path).WithField("unit", string(durationDescription[len(durationDescription)-1])).Warning("possible input-file does not match pattern 'forecast_*(h|d)_ahead.csv', where * is a non-negative integer (illegal duration-unit)")
				return nil
			}

			number, err := strconv.Atoi(durationDescription[:len(durationDescription)-1])
			if err != nil {
				log.WithField("filepath", path).WithError(err).Warning("possible input-file does not match pattern 'forecast_*(h|d)_ahead.csv', where * is a non-negative integer (illegal duration-number)")
				return nil
			}

			files[time.Duration(number)*multiplier] = fileInfo{
				path:     path,
				isHourly: multiplier == time.Hour,
			}
			return nil
		}
		return nil
	})
	if err != nil {
		return nil, nil, nil, nil, err
	}

	wd := make(map[time.Duration]map[time.Time]*weather.Data)
	oldm := make(map[time.Duration]*time.Time)
	latestm := make(map[time.Duration]*time.Time)

	for d, fi := range files {
		log.WithField("filepath", fi.path).WithField("isHourly", fi.isHourly).WithField("duration_ahead", d).Debug("reading next weather-input-file")
		f, err := os.OpenFile(fi.path, os.O_RDONLY, os.ModePerm)
		if err != nil {
			return nil, nil, nil, nil, err
		}
		defer f.Close()

		type weatherCSVData struct {
			*weather.Data
			Time time.Time `csv:"Time"`
		}
		ws := []*weatherCSVData{}

		if err := gocsv.UnmarshalFile(f, &ws); err != nil {
			return nil, nil, nil, nil, err
		}
		wd[d] = make(map[time.Time]*weather.Data)

		var oldest *time.Time
		var latest *time.Time

		log.WithField("amount", len(ws)).Debug("elements extracted")

		for _, w := range ws {
			log.WithField("element", w).Trace()
			r := round(w.Time)
			if wd[d][r] == nil || fi.isHourly {
				wd[d][r] = w.Data
				if oldest == nil || oldest.Sub(r) > 0 {
					oldest = &r
				}
				if latest == nil || latest.Sub(r) < 0 {
					latest = &r
				}
			}
		}
		log.WithField("oldest", oldest).WithField("latest", latest).Debug("file processed")
		oldm[d] = oldest
		latestm[d] = latest
	}

	forecastPoints = make([]time.Duration, 0)
	maxForecastDistance := time.Duration(stepAmount-1) * config.Viper.GetDuration(PathStepSize)
	for i := 0 * time.Second; i <= maxForecastDistance; i += time.Hour {
		forecastPoints = append(forecastPoints, i)
	}

	log.WithField("amount_forecastPoints", len(forecastPoints)).Info("weather-processing complete")
	for _, d := range forecastPoints {
		log.WithField("forecastPoint", d).WithField("backed", wd[d] != nil).Debug()
	}

	return oldm, latestm, forecastPoints, wd, nil
}

func readProductionInput(productionAddress string, round func(time.Time) time.Time) (oldest, latest *time.Time, data map[time.Time]*production.Data, err error) {
	log.Info("reading production data...")

	f, err := os.OpenFile(productionAddress, os.O_RDONLY, os.ModePerm)
	if err != nil {
		return nil, nil, nil, err
	}
	defer f.Close()

	type prodCSVData struct {
		*production.Data
		Time time.Time `csv:"Time"`
	}
	ps := []*prodCSVData{}

	if err := gocsv.UnmarshalFile(f, &ps); err != nil {
		return nil, nil, nil, err
	}

	pd := make(map[time.Time]*production.Data)

	for _, p := range ps {
		r := round(p.Time)
		if pd[r] == nil {
			pd[r] = p.Data
			if oldest == nil || oldest.Sub(r) > 0 {
				oldest = &r
			}
			if latest == nil || latest.Sub(r) < 0 {
				latest = &r
			}
		} else {
			log.WithField("timestep", r).Warn("conflicting production input")
		}
	}

	if len(pd) == 0 {
		log.Warning("no production data found")
	} else {
		log.WithField("amount", len(pd)).Info("production-processing complete")
	}

	return oldest, latest, pd, nil
}
