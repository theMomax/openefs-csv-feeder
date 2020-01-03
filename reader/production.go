package reader

import (
	"time"

	model "github.com/theMomax/openefs/models/production"
)

func (r *Reader) ReadProduction(date time.Time, replace ...bool) *model.Data {
	if len(r.production) == 0 || r.oldestProductionData.Sub(date) > 0 {
		return nil
	}

	if val := r.production[r.round(date)]; val != nil {
		return val
	}
	if len(replace) == 1 && replace[0] {
		return r.ReadProduction(date.Add(-1*r.productionTimestep), true)
	}
	return nil
}
