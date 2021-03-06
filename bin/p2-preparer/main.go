package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/Sirupsen/logrus"
	"github.com/square/p2/pkg/logging"
	"github.com/square/p2/pkg/preparer"
	"github.com/square/p2/pkg/version"
)

func main() {
	logger := logging.NewLogger(logrus.Fields{})
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		logger.NoFields().Fatalln("No CONFIG_PATH variable was given")
	}
	preparerConfig, err := preparer.LoadPreparerConfig(configPath)
	if err != nil {
		logger.WithField("inner_err", err).Fatalln("could not load preparer config")
	}

	if preparerConfig.KeyringPath == "" {
		logger.NoFields().Fatalln("The preparer must be launched with a keyring")
	}

	prep, err := preparer.New(preparerConfig, logger)
	if err != nil {
		logger.WithField("inner_err", err).Fatalln("Could not initialize preparer")
	}

	logger.WithFields(logrus.Fields{
		"starting":  true,
		"node_name": preparerConfig.NodeName,
		"consul":    preparerConfig.ConsulAddress,
		"hooks_dir": preparerConfig.HooksDirectory,
		"keyring":   preparerConfig.KeyringPath,
		"version":   version.VERSION,
	}).Infoln("Preparer started successfully")

	quitMainUpdate := make(chan struct{})
	quitHookUpdate := make(chan struct{})
	go prep.WatchForPodManifestsForNode(quitMainUpdate)
	go prep.WatchForHooks(quitHookUpdate)

	waitForTermination(logger, quitMainUpdate, quitHookUpdate)

	logger.NoFields().Infoln("Terminating")
}

func waitForTermination(logger logging.Logger, quitMainUpdate, quitHookUpdate chan struct{}) {
	signalCh := make(chan os.Signal, 2)
	signal.Notify(signalCh, syscall.SIGTERM, os.Interrupt)
	received := <-signalCh
	logger.WithField("signal", received.String()).Infoln("Stopping work")
	quitHookUpdate <- struct{}{}
	quitMainUpdate <- struct{}{}
	<-quitMainUpdate // acknowledgement
}
