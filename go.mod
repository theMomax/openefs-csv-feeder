module github.com/theMomax/openefs-csv-feeder

go 1.13

replace github.com/theMomax/openefs => ../openefs

require (
	github.com/gocarina/gocsv v0.0.0-20191214001331-e6697589f2e0
	github.com/jonboulle/clockwork v0.1.0
	github.com/sirupsen/logrus v1.4.2
	github.com/spf13/cobra v0.0.5
	github.com/spf13/viper v1.5.0
	github.com/theMomax/openefs v0.0.0-20191208114622-879256546465
)
