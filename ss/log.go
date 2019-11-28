package ss

import (
	"log"
	"os"
)

var isDebug bool
var null, _ = os.Open(os.DevNull)
var Debug = log.New(null, "[DEBUG]", log.Ltime|log.Lshortfile)
