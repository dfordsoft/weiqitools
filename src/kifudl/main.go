package main

import (
	"fmt"
	"kifudl/hoetom"
	"kifudl/lol"
	"kifudl/onegreen"
	"kifudl/sina"
	"kifudl/tom"
	"kifudl/weiqitv"
	"kifudl/xgoo"
	"sync"

	flag "github.com/ogier/pflag"
)

var (
	wg               sync.WaitGroup
	quitIfExists     bool
	saveFileEncoding string
	parallelCount    int
	downloadCount    int32
)

func main() {
	flag.StringVar(&saveFileEncoding, "encoding", "gbk", "save SGF file encoding")
	flag.BoolVar(&quitIfExists, "q", true, "quit if the target file exists")
	flag.IntVar(&parallelCount, "p", 20, "the parallel routines count")

	h := &hoetom.Hoetom{}
	flag.IntVar(&h.LatestPageID, "hoetom-latest-page-id", 1, "the latest page id of hoetom")
	flag.IntVar(&h.EarliestPageID, "hoetom-earliest-page-id", 1045, "the earliest page id of hoetom")

	flag.Parse()

	fmt.Println("save SGF file encoding", saveFileEncoding)
	fmt.Println("quit if the target file exists", quitIfExists)
	fmt.Println("the parallel routines count", parallelCount)

	h.SaveFileEncoding = saveFileEncoding
	h.QuitIfExists = quitIfExists
	h.ParallelCount = parallelCount

	go lol.Download(&wg)
	go h.Download(&wg)
	go sina.Download(&wg)
	go tom.Download(&wg)
	go xgoo.Download(&wg)
	go weiqitv.Download(&wg)
	go onegreen.Download(&wg)

	wg.Wait()
}
