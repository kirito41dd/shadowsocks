package ss

import (
	"github.com/zshorz/ezlog"
	"os"
)

var null, _ = os.Open(os.DevNull)
var Debug = ezlog.New(os.Stdout, "", ezlog.BitDefault, ezlog.LogAll)

func init() {
	SetDebug(true)
}
