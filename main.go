package main

import (
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// conf represents the configuration passed to the application via command
// line options, environment variables, and the configuration file.
var conf *viper.Viper

// logger represents the configuration passed to the application via command
// line options, environment variables, and the configuration file.
var logger *Logger

// init initializes the configuration and logging.
func init() {

	conf = viper.New()

	conf.SetEnvPrefix("FIFO2KINESIS")
	conf.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	conf.AutomaticEnv()

	viper.SetConfigName("fifo2kinesis")

	pflag.BoolP("debug", "d", false, "Show debug level log messages")
	conf.BindPFlag("debug", pflag.Lookup("debug"))
	conf.SetDefault("debug", "")

	pflag.StringP("fifo-name", "f", "", "The absolute path of the named pipe, e.g. /var/test.pipe")
	conf.BindPFlag("fifo-name", pflag.Lookup("fifo-name"))
	conf.SetDefault("fifo-name", "")

	pflag.StringP("partition-key", "p", "", "The partition key, defaults to a 12 character random string if omitted")
	conf.BindPFlag("partition-key", pflag.Lookup("partition-key"))
	conf.SetDefault("partition-key", "")

	pflag.StringP("stream-name", "s", "", "The name of the Kinesis stream")
	conf.BindPFlag("stream-name", pflag.Lookup("stream-name"))
	conf.SetDefault("stream-name", "")

	pflag.Parse()

	if conf.GetBool("debug") {
		logger = NewLogger(LOG_DEBUG)
	} else {
		logger = NewLogger(LOG_INFO)
	}

	logger.Debug("configuration parsed")
}

func main() {

	fn := conf.GetString("fifo-name")
	if fn == "" {
		logger.Fatal("missing required option: fifo-name")
	}

	sn := conf.GetString("stream-name")
	if sn == "" {
		logger.Fatal("missing required option: stream-name")
	}

	pk := conf.GetString("partition-key")
	StartPipeline(NewFifo(fn, sn, pk))
}

// StartPipeline sets up the event handler continuously runs the pipeline,
// i.e. reads data from the FIFO and published data records to Kinesis.
func StartPipeline(fifo *Fifo) {
	logger.Notice("starting pipeline")

	stop := make(chan bool)
	go EventListener(stop)

	go func() {
		if err := fifo.RunPipeline(); err != nil {
			logger.Fatal(err)
		}
	}()

	<-stop
	logger.Notice("pipeline stopped")
}

// EventListener listens for signals in order to stop the application.
func EventListener(stop chan bool) {
	quit := make(chan os.Signal)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	for {
		select {
		case <-quit:
			logger.Notice("stopping pipeline")
			stop <- true
			break
		}
	}
}
