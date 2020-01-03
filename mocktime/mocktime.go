package mocktime

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jonboulle/clockwork"
	"github.com/theMomax/openefs-csv-feeder/config"
)

// Config paths
const (
	PathMockTimeAddress = "mocktime.address"
)

var fakeClock clockwork.FakeClock

func init() {
	config.RootCtx.PersistentFlags().StringP(PathMockTimeAddress, "m", "http://localhost:8090", "address for mock-time endpoint")
	config.Viper.BindPFlag(PathMockTimeAddress, config.RootCtx.PersistentFlags().Lookup(PathMockTimeAddress))
}

func Update(t time.Time) error {
	resp, err := http.Get(config.Viper.GetString(PathMockTimeAddress) + strings.ReplaceAll("/utils/time/mocktime/:unixtimestamp", ":unixtimestamp", strconv.FormatInt(t.Unix(), 10)))
	if err != nil {
		return errors.New("mock-time update failed: " + err.Error())
	} else if resp.StatusCode != http.StatusOK {
		return errors.New("mock-time update failed: " + strconv.Itoa(resp.StatusCode) + "; " + resp.Status)
	}
	return nil
}
