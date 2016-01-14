package main

import (
	"os"
	"os/signal"
	"strings"
	"sync"
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

	pflag.String("fifo-name", "f", "The absolute path of the named pipe, e.g. /var/test.pipe")
	conf.BindPFlag("fifo-name", pflag.Lookup("fifo-name"))
	conf.SetDefault("fifo-name", "")

	pflag.StringP("stream-name", "s", "", "The name of the Kinesis stream")
	conf.BindPFlag("stream-name", pflag.Lookup("stream-name"))
	conf.SetDefault("stream-name", "")

	pflag.BoolP("debug", "d", false, "Show debug level log messages")
	conf.BindPFlag("debug", pflag.Lookup("debug"))
	conf.SetDefault("debug", "")

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

	StartPipeline(NewFifo(fn, sn))
}

// StartPipeline sets up the event handler continuously runs the pipeline,
// i.e. reads data from the FIFO and published data records to Kinesis.
func StartPipeline(fifo *Fifo) {
	logger.Notice("starting pipeline")

	wg := &sync.WaitGroup{}
	stop := make(chan bool)

	go EventListener(wg, stop)

	for {
		select {
		case <-stop:
			wg.Wait()
			logger.Notice("pipeline stopped")
			return

		default:
			if err := fifo.RunPipeline(); err != nil {
				logger.Fatal(err)
			}
		}
	}
}

// EventListener listens for signals in order to stop the application.
func EventListener(wg *sync.WaitGroup, stop chan bool) {
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
