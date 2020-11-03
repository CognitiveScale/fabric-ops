package main

import (
	"fabric-ops/cmd"
)

var Version = "NA" // this will be set in build. see makefile

func main() {
	//log.SetFlags(log.LstdFlags | log.Lshortfile)
	cmd.Execute(Version)
}
