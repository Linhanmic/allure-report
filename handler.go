package main

import (
	"context"
	"os"

	"github.com/getgauge/gauge-proto/go/gauge_messages"
	"google.golang.org/grpc"
)

type handler struct {
	gauge_messages.UnimplementedReporterServer
	server *grpc.Server
}

func (*handler) NotifyExecutionStarting(context.Context, *gauge_messages.ExecutionStartingRequest) (*gauge_messages.Empty, error) {
	return &gauge_messages.Empty{}, nil
}

func (*handler) NotifySpecExecutionStarting(context.Context, *gauge_messages.SpecExecutionStartingRequest) (*gauge_messages.Empty, error) {
	return &gauge_messages.Empty{}, nil
}

func (*handler) NotifyScenarioExecutionStarting(context.Context, *gauge_messages.ScenarioExecutionStartingRequest) (*gauge_messages.Empty, error) {
	return &gauge_messages.Empty{}, nil
}

func (*handler) NotifyConceptExecutionStarting(context.Context, *gauge_messages.ConceptExecutionStartingRequest) (*gauge_messages.Empty, error) {
	return &gauge_messages.Empty{}, nil
}

func (*handler) NotifyConceptExecutionEnding(context.Context, *gauge_messages.ConceptExecutionEndingRequest) (*gauge_messages.Empty, error) {
	return &gauge_messages.Empty{}, nil
}

func (*handler) NotifyStepExecutionStarting(context.Context, *gauge_messages.StepExecutionStartingRequest) (*gauge_messages.Empty, error) {
	return &gauge_messages.Empty{}, nil
}

func (*handler) NotifyStepExecutionEnding(context.Context, *gauge_messages.StepExecutionEndingRequest) (*gauge_messages.Empty, error) {
	return &gauge_messages.Empty{}, nil
}

func (*handler) NotifyScenarioExecutionEnding(context.Context, *gauge_messages.ScenarioExecutionEndingRequest) (*gauge_messages.Empty, error) {
	return &gauge_messages.Empty{}, nil
}

func (*handler) NotifySpecExecutionEnding(context.Context, *gauge_messages.SpecExecutionEndingRequest) (*gauge_messages.Empty, error) {
	return &gauge_messages.Empty{}, nil
}

func (*handler) NotifyExecutionEnding(context.Context, *gauge_messages.ExecutionEndingRequest) (*gauge_messages.Empty, error) {
	return &gauge_messages.Empty{}, nil
}

func (*handler) NotifySuiteResult(_ context.Context, m *gauge_messages.SuiteExecutionResult) (*gauge_messages.Empty, error) {
	createReport(m)
	return &gauge_messages.Empty{}, nil
}

func (h *handler) Kill(context.Context, *gauge_messages.KillProcessRequest) (*gauge_messages.Empty, error) {
	defer h.stopServer()
	return &gauge_messages.Empty{}, nil
}

func (h *handler) stopServer() {
	h.server.Stop()
	os.Exit(0)
}
