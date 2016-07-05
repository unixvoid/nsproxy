package nslog

import (
	"io"
	"log"
)

var (
	// initialize logger types
	Info    *log.Logger
	Cluster *log.Logger
	Debug   *log.Logger
	Error   *log.Logger
)

func LogInit(infoHandler, clusterHandler, debugHandler, errorHandler io.Writer) {
	// read loglevel
	Info = log.New(infoHandler,
		"INFO: ",
		log.Ldate|log.Ltime)
	Cluster = log.New(clusterHandler,
		"CLUSTER: ",
		log.Ldate|log.Ltime|log.Lshortfile)
	Debug = log.New(debugHandler,
		"DEBUG: ",
		log.Ldate|log.Ltime|log.Lshortfile)
	Error = log.New(errorHandler,
		"ERROR: ",
		log.Ldate|log.Ltime|log.Lshortfile)
}
