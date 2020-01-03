package reader

import (
	"time"

	"github.com/theMomax/openefs/models/production"
	"github.com/theMomax/openefs/models/production/weather"
)

type Iterator struct {
	reader  *Reader
	isEmpty bool
	curr    time.Time
	end     time.Time
}

func (r *Reader) NewIterator(start ...time.Time) *Iterator {
	if r.oldestProductionData == nil || r.oldestWeatherData == nil {
		return &Iterator{
			isEmpty: true,
		}
	}

	s := *r.oldestProductionData
	for _, t := range r.oldestWeatherData {
		if t != nil && s.Sub(*t) > 0 {
			s = *t
		}
	}

	if len(start) == 1 && s.Sub(start[0]) < 0 {
		s = start[0]
	}

	end := *r.latestProductionData
	for _, t := range r.latestWeatherData {
		if t != nil && s.Sub(*t) < 0 {
			end = *t
		}
	}

	return &Iterator{
		reader: r,
		curr:   s,
		end:    end,
	}
}

func (r *Reader) ForEach(productionCallback func(time.Time, *production.Data), weatherCallback func([]time.Time, []*weather.Data), start ...time.Time) {
	it := r.NewIterator(start...)
	for it.HasNext() {
		t, p, w := it.Next(true, true)
		productionCallback(t, p)
		times := make([]time.Time, len(w))
		for i := range times {
			times[i] = t.Add(time.Duration(i) * time.Hour)
		}
		weatherCallback(times, w)
	}
}

func (i *Iterator) HasNext() bool {
	return !i.isEmpty && i.curr.Sub(i.end) <= 0
}

// Next returns the next production- and weather-data. If there is none the
// function returnes (nil, nil). If replace[0] is set, the function is going to
// use more recent forecast data for the weather array. If replace[1] is set,
// previous time-steps are used for replacement for both production and weather
// if necessary.
func (i *Iterator) Next(replace ...bool) (time.Time, *production.Data, []*weather.Data) {
	if !i.HasNext() {
		return time.Unix(0, 0), nil, nil
	}

	replaceByMoreRecentForecast := len(replace) >= 1 && replace[0]
	replaceByOlderTimestamp := len(replace) >= 2 && replace[1]

	p := i.reader.ReadProduction(i.curr, replaceByOlderTimestamp)
	w := i.reader.ReadWeatherForecast(i.curr, replaceByMoreRecentForecast, replaceByOlderTimestamp)
	c := i.curr
	i.curr = i.curr.Add(i.reader.productionTimestep)
	return c, p, w
}
