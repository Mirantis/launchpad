// +build windows

package main

import (
	"os"

	"github.com/shiena/ansicolor"
	log "github.com/sirupsen/logrus"
)

func init() {
	log.SetFormatter(&log.TextFormatter{ForceColors: true})
	log.SetOutput(ansicolor.NewAnsiColorWriter(os.Stdout))
}
