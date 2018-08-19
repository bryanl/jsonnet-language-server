package main

import (
	"context"
	"encoding/json"
	"flag"
	"os"
	"sync"

	"github.com/bryanl/jsonnet-language-server/pkg/server"

	"github.com/sourcegraph/jsonrpc2"

	"github.com/sirupsen/logrus"
)

func main() {
	var debug bool
	flag.BoolVar(&debug, "debug", false, "enable debug logging")
	flag.Parse()

	logger := initLogger(debug)

	if err := run(logger); err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

	logger.Info("exiting")
}

func run(logger logrus.FieldLogger) error {
	logger.Info("scanning stdin")

	handler := server.NewHandler(logger)

	logOpt := LogMessages(logger)
	<-jsonrpc2.NewConn(context.Background(), jsonrpc2.NewBufferedStream(stdrwc{}, jsonrpc2.VSCodeObjectCodec{}), handler, logOpt).DisconnectNotify()

	return nil
}

func initLogger(debug bool) logrus.FieldLogger {
	logger := logrus.New()
	logger.Formatter = &logrus.TextFormatter{}

	// TODO set up an option to configure logging to a file or stderr
	// logName := filepath.Join("/tmp", "jsp-"+time.Now().Format("20060102150405")+".log")
	// logger.WithField("log-path", logName).Info("configuring log output")

	// f, err := os.OpenFile(logName, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
	// if err != nil {
	// 	logger.WithError(err).Fatal("unable to open log file")
	// }

	// logger.SetOutput(f)
	// logrus.SetOutput(f)

	if debug {
		logger.SetLevel(logrus.DebugLevel)
		logrus.SetLevel(logrus.DebugLevel)
	}

	return logger.WithFields(logrus.Fields{
		"app": "jsonnet-language-server",
	})
}

type stdrwc struct{}

func (stdrwc) Read(p []byte) (int, error) {
	return os.Stdin.Read(p)
}

func (stdrwc) Write(p []byte) (int, error) {
	return os.Stdout.Write(p)
}

func (stdrwc) Close() error {
	if err := os.Stdin.Close(); err != nil {
		return err
	}
	return os.Stdout.Close()
}

// LogMessages causes all messages sent and received on conn to be
// logged using the provided logger.
func LogMessages(log logrus.FieldLogger) jsonrpc2.ConnOpt {
	return func(c *jsonrpc2.Conn) {
		// Remember reqs we have received so we can helpfully show the
		// request method in OnSend for responses.
		var (
			mu         sync.Mutex
			reqMethods = map[jsonrpc2.ID]string{}
		)

		jsonrpc2.OnRecv(func(req *jsonrpc2.Request, resp *jsonrpc2.Response) {
			switch {
			case req != nil && resp == nil:
				mu.Lock()
				reqMethods[req.ID] = req.Method
				mu.Unlock()

				params, _ := json.Marshal(req.Params)
				if req.Notif {
					log.Printf("--> notif: %s: %s", req.Method, params)
				} else {
					log.Printf("--> request #%s: %s: %s", req.ID, req.Method, params)
				}

			case resp != nil:
				var method string
				if req != nil {
					method = req.Method
				} else {
					method = "(no matching request)"
				}
				switch {
				case resp.Result != nil:
					result, _ := json.Marshal(resp.Result)
					log.Printf("--> result #%s: %s: %s", resp.ID, method, result)
				case resp.Error != nil:
					err, _ := json.Marshal(resp.Error)
					log.Printf("--> error #%s: %s: %s", resp.ID, method, err)
				}
			}
		})(c)
		jsonrpc2.OnSend(func(req *jsonrpc2.Request, resp *jsonrpc2.Response) {
			switch {
			case req != nil:
				params, _ := json.Marshal(req.Params)
				if req.Notif {
					log.Printf("<-- notif: %s: %s", req.Method, params)
				} else {
					log.Printf("<-- request #%s: %s: %s", req.ID, req.Method, params)
				}

			case resp != nil:
				mu.Lock()
				method := reqMethods[resp.ID]
				delete(reqMethods, resp.ID)
				mu.Unlock()
				if method == "" {
					method = "(no previous request)"
				}

				if resp.Result != nil {
					result, _ := json.Marshal(resp.Result)
					log.Printf("<-- result #%s: %s: %s", resp.ID, method, result)
				} else {
					err, _ := json.Marshal(resp.Error)
					log.Printf("<-- error #%s: %s: %s", resp.ID, method, err)
				}
			}
		})(c)
	}
}
