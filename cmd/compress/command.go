package compress

import (
	log "github.com/sirupsen/logrus"
)

type Command struct{}

// Execute current command.
func (*Command) Execute(_ []string) (err error) {
	var logger = log.New()

	// set logger properties
	logger.SetFormatter(&log.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "15:04:05.000",
	})

	logger.Info("Not implemented yet")

	return
}
