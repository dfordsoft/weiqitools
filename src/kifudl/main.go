package main

import (
	"fmt"
	"kifudl/hoetom"
	"kifudl/lol"
	"kifudl/onegreen"
	"kifudl/semaphore"
	"kifudl/sina"
	"kifudl/tom"
	"kifudl/weiqitv"
	"kifudl/xgoo"
	"sync"

	flag "github.com/ogier/pflag"
)

func main() {
	var quitIfExists bool
	var saveFileEncoding string
	var parallelCount int
	flag.StringVar(&saveFileEncoding, "encoding", "gbk", "save SGF file encoding")
	flag.BoolVar(&quitIfExists, "q", true, "quit if the target file exists")
	flag.IntVar(&parallelCount, "p", 20, "the parallel routines count")

	var hoetomLatestPageID, hoetomEarliestPageID int
	flag.IntVar(&hoetomLatestPageID, "hoetom-latest-page-id", 1, "the latest page id of hoetom")
	flag.IntVar(&hoetomEarliestPageID, "hoetom-earliest-page-id", 1045, "the earliest page id of hoetom")

	var lolLatestID, lolEarliestID int
	flag.IntVar(&lolLatestID, "lol-latest-id", 0, "the latest pid of 101weiqi")
	flag.IntVar(&lolEarliestID, "lol-earliest-id", 1, "the earliest pid of 101weiqi")

	var sinaLatestPageID, sinaEarliestPageID int
	flag.IntVar(&sinaLatestPageID, "sina-latest-page-id", 0, "the latest page id of sina")
	flag.IntVar(&sinaEarliestPageID, "sina-earliest-page-id", 689, "the earliest page id of sina")

	var xgooLatestPageID, xgooEarliestPageID int
	flag.IntVar(&xgooLatestPageID, "xgoo-latest-page-id", 1, "the latest page id of xgoo")
	flag.IntVar(&xgooEarliestPageID, "xgoo-earliest-page-id", 1968, "the earliest page id of xgoo")

	var weiqitvStartID, weiqitvEndID int
	flag.IntVar(&weiqitvStartID, "weiqitv-start-id", 0, "the start id")
	flag.IntVar(&weiqitvEndID, "weiqitv-end-id", 77281, "the end id")

	flag.Parse()

	fmt.Println("save SGF file encoding", saveFileEncoding)
	fmt.Println("quit if the target file exists", quitIfExists)
	fmt.Println("the parallel routines count", parallelCount)

	h := &hoetom.Hoetom{
		Semaphore:        *semaphore.NewSemaphore(parallelCount),
		SaveFileEncoding: saveFileEncoding,
		QuitIfExists:     quitIfExists,
		LatestPageID:     hoetomLatestPageID,
		EarliestPageID:   hoetomEarliestPageID,
	}

	l := &lol.Lol{
		Semaphore:        *semaphore.NewSemaphore(parallelCount),
		SaveFileEncoding: saveFileEncoding,
		QuitIfExists:     quitIfExists,
		LatestID:         lolLatestID,
		EarliestID:       lolEarliestID,
	}

	s := &sina.Sina{
		Semaphore:        *semaphore.NewSemaphore(parallelCount),
		SaveFileEncoding: saveFileEncoding,
		QuitIfExists:     quitIfExists,
		LatestPageID:     sinaLatestPageID,
		EarliestPageID:   sinaEarliestPageID,
	}

	x := &xgoo.Xgoo{
		Semaphore:        *semaphore.NewSemaphore(parallelCount),
		SaveFileEncoding: saveFileEncoding,
		QuitIfExists:     quitIfExists,
		LatestPageID:     xgooLatestPageID,
		EarliestPageID:   xgooEarliestPageID,
	}

	w := &weiqitv.WeiqiTV{
		Semaphore:        *semaphore.NewSemaphore(parallelCount),
		SaveFileEncoding: saveFileEncoding,
		QuitIfExists:     quitIfExists,
		StartID:          weiqitvStartID,
		EndID:            weiqitvEndID,
	}

	o := &onegreen.Onegreen{
		Semaphore:        *semaphore.NewSemaphore(parallelCount),
		SaveFileEncoding: saveFileEncoding,
		QuitIfExists:     quitIfExists,
	}

	t := &tom.Tom{
		Semaphore:        *semaphore.NewSemaphore(parallelCount),
		SaveFileEncoding: saveFileEncoding,
		QuitIfExists:     quitIfExists,
	}

	var wg sync.WaitGroup
	wg.Add(7)
	go l.Download(&wg)
	go h.Download(&wg)
	go s.Download(&wg)
	go t.Download(&wg)
	go x.Download(&wg)
	go w.Download(&wg)
	go o.Download(&wg)

	wg.Wait()
	fmt.Println("total downloaded ",
		l.DownloadCount+h.DownloadCount+s.DownloadCount+t.DownloadCount+x.DownloadCount+w.DownloadCount+o.DownloadCount,
		" SGFs")
}
