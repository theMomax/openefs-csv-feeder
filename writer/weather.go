package writer

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	models "github.com/theMomax/openefs/models/production/weather"
)

func (w *Writer) WriteWeather(date time.Time, data *models.Data) error {
	b, err := json.Marshal(data)
	if err != nil {
		return err
	}

	for {
		log.WithField("data", data).WithField("date", date).Trace("trying to send weather-data...")
		resp, err := http.Post(strings.ReplaceAll(w.weatheraddress, ":unixtimestamp", strconv.FormatInt(date.Unix(), 10)), "application/json", bytes.NewReader(b))
		if err != nil {
			return err
		}

		switch resp.StatusCode {
		case http.StatusOK:
			log.WithField("data", data).WithField("date", date).Debug("succesfully sent weather-data")
			return nil
		case http.StatusIMUsed:
			time.Sleep(time.Second)
		default:
			return errors.New(resp.Status)
		}
	}
}
