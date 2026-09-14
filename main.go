package main

import (
	"net"
	"os"

	"github.com/Linhanmic/allure-report/logger"
	gm "github.com/getgauge/gauge-proto/go/gauge_messages"
	"google.golang.org/grpc"
)

const oneGB = 1024 * 1024 * 1024

func main() {
	findPluginAndProjectRoot()
	if os.Getenv(pluginActionEnv) != executionAction {
		return
	}
	if err := os.Chdir(projectRoot); err != nil {
		logger.Fatal("failed to change directory to project root: %s", err)
	}
	address, err := net.ResolveTCPAddr("tcp", "127.0.0.1:0")
	if err != nil {
		logger.Fatal("failed to start server.")
	}
	l, err := net.ListenTCP("tcp", address)
	if err != nil {
		logger.Fatal("failed to start server.")
	}
	server := grpc.NewServer(grpc.MaxRecvMsgSize(oneGB))
	h := &handler{server: server}
	gm.RegisterReporterServer(server, h)
	logger.Info("Listening on port:%d", l.Addr().(*net.TCPAddr).Port)
	if err := server.Serve(l); err != nil {
		logger.Fatal("failed to serve: %s", err)
	}
}
