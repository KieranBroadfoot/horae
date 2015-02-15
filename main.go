package main // import "github.com/kieranbroadfoot/horae"

import (
	"flag"
	"github.com/kieranbroadfoot/horae/core"
)

func main() {
	clusterName := flag.String("clustername", "default", "The horae cluster name")
	flag.Parse()

	core.StartServer(*clusterName)
}
