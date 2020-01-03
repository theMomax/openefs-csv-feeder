package reader

import (
	"time"

	model "github.com/theMomax/openefs/models/production/weather"
)

func (r *Reader) ReadWeather(date time.Time, distance time.Duration, replace ...bool) *model.Data {
	if r.oldestWeatherData[distance] == nil || r.weather[distance] == nil {
		return nil
	}

	if len(r.production) == 0 || r.oldestWeatherData[distance].Sub(date) > 0 {
		return nil
	}

	if val := r.weather[distance][r.round(date)]; val != nil {
		return val
	}
	if len(replace) == 1 && replace[0] {
		return r.ReadWeather(date.Add(-1*r.productionTimestep), distance, true)
	}
	return nil
}

func (r *Reader) ReadWeatherForecast(date time.Time, replace ...bool) []*model.Data {
	replaceByMoreRecentForecast := len(replace) >= 1 && replace[0]
	replaceByOlderTimestamp := len(replace) >= 2 && replace[1]

	vals := make([]*model.Data, len(r.forecastPoints))
	for i, d := range r.forecastPoints {
		t := date.Add(d)
		vals[i] = r.ReadWeather(t, d, replaceByOlderTimestamp)
		d -= time.Hour
		for vals[i] == nil && d >= 0 && replaceByMoreRecentForecast {
			vals[i] = r.ReadWeather(t, d, replaceByOlderTimestamp)
			d -= time.Hour
		}
	}
	return vals
}
