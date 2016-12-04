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

	l := &lol.Lol{}
	flag.IntVar(&l.LatestID, "lol-latest-id", 0, "the latest pid of 101weiqi")
	flag.IntVar(&l.EarliestID, "lol-earliest-id", 1, "the earliest pid of 101weiqi")

	s := &sina.Sina{}
	flag.IntVar(&s.LatestPageID, "sina-latest-page-id", 0, "the latest page id of sina")
	flag.IntVar(&s.EarliestPageID, "sina-earliest-page-id", 689, "the earliest page id of sina")

	x := &xgoo.Xgoo{}
	flag.IntVar(&x.LatestPageID, "xgoo-latest-page-id", 1, "the latest page id of xgoo")
	flag.IntVar(&x.EarliestPageID, "xgoo-earliest-page-id", 1968, "the earliest page id of xgoo")

	w := &weiqitv.WeiqiTV{}
	flag.IntVar(&w.StartID, "weiqitv-start-id", 0, "the start id")
	flag.IntVar(&w.EndID, "weiqitv-end-id", 77281, "the end id")

	flag.Parse()

	fmt.Println("save SGF file encoding", saveFileEncoding)
	fmt.Println("quit if the target file exists", quitIfExists)
	fmt.Println("the parallel routines count", parallelCount)

	h.SaveFileEncoding = saveFileEncoding
	h.QuitIfExists = quitIfExists
	h.Sem = semaphore.NewSemaphore(parallelCount)

	l.SaveFileEncoding = saveFileEncoding
	l.QuitIfExists = quitIfExists
	l.Sem = semaphore.NewSemaphore(parallelCount)

	s.SaveFileEncoding = saveFileEncoding
	s.QuitIfExists = quitIfExists
	s.Sem = semaphore.NewSemaphore(parallelCount)

	x.SaveFileEncoding = saveFileEncoding
	x.QuitIfExists = quitIfExists
	x.Sem = semaphore.NewSemaphore(parallelCount)

	w.SaveFileEncoding = saveFileEncoding
	w.QuitIfExists = quitIfExists
	w.Sem = semaphore.NewSemaphore(parallelCount)

	o := &onegreen.Onegreen{
		Sem:              semaphore.NewSemaphore(parallelCount),
		SaveFileEncoding: saveFileEncoding,
		QuitIfExists:     quitIfExists,
	}

	t := &tom.Tom{
		Sem:              semaphore.NewSemaphore(parallelCount),
		SaveFileEncoding: saveFileEncoding,
		QuitIfExists:     quitIfExists,
	}

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
