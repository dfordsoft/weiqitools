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
	var lolEnabled, xgooEnabled, sinaEnabled, tomEnabled, onegreenEnabled, hoetomEnabled, weiqitvEnabled bool
	flag.StringVar(&saveFileEncoding, "encoding", "gbk", "save SGF file encoding")
	flag.BoolVar(&quitIfExists, "q", true, "quit if the target file exists")
	flag.IntVar(&parallelCount, "p", 20, "the parallel routines count")
	flag.BoolVar(&lolEnabled, "lol-enabled", true, "fetch kifu from lol")
	flag.BoolVar(&xgooEnabled, "xgoo-enabled", true, "fetch kifu from xgoo")
	flag.BoolVar(&sinaEnabled, "sina-enabled", true, "fetch kifu from sina")
	flag.BoolVar(&tomEnabled, "tom-enabled", true, "fetch kifu from tom")
	flag.BoolVar(&onegreenEnabled, "onegreen-enabled", true, "fetch kifu from onegreen")
	flag.BoolVar(&hoetomEnabled, "hoetom-enabled", true, "fetch kifu from hoetom")
	flag.BoolVar(&weiqitvEnabled, "weiqitv-enabled", true, "fetch kifu from weiqitv")

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

	var wg sync.WaitGroup

	var h *hoetom.Hoetom
	if hoetomEnabled {
		h := &hoetom.Hoetom{
			Semaphore:        *semaphore.NewSemaphore(parallelCount),
			SaveFileEncoding: saveFileEncoding,
			QuitIfExists:     quitIfExists,
			LatestPageID:     hoetomLatestPageID,
			EarliestPageID:   hoetomEarliestPageID,
		}
		wg.Add(1)
		go h.Download(&wg)
	}

	var l *lol.Lol
	if lolEnabled {
		l := &lol.Lol{
			Semaphore:        *semaphore.NewSemaphore(parallelCount),
			SaveFileEncoding: saveFileEncoding,
			QuitIfExists:     quitIfExists,
			LatestID:         lolLatestID,
			EarliestID:       lolEarliestID,
		}
		wg.Add(1)
		go l.Download(&wg)
	}

	var s *sina.Sina
	if sinaEnabled {
		s := &sina.Sina{
			Semaphore:        *semaphore.NewSemaphore(parallelCount),
			SaveFileEncoding: saveFileEncoding,
			QuitIfExists:     quitIfExists,
			LatestPageID:     sinaLatestPageID,
			EarliestPageID:   sinaEarliestPageID,
		}
		wg.Add(1)
		go s.Download(&wg)
	}

	var x *xgoo.Xgoo
	if xgooEnabled {
		x := &xgoo.Xgoo{
			Semaphore:        *semaphore.NewSemaphore(parallelCount),
			SaveFileEncoding: saveFileEncoding,
			QuitIfExists:     quitIfExists,
			LatestPageID:     xgooLatestPageID,
			EarliestPageID:   xgooEarliestPageID,
		}
		wg.Add(1)
		go x.Download(&wg)
	}

	var w *weiqitv.WeiqiTV
	if weiqitvEnabled {
		w := &weiqitv.WeiqiTV{
			Semaphore:        *semaphore.NewSemaphore(parallelCount),
			SaveFileEncoding: saveFileEncoding,
			QuitIfExists:     quitIfExists,
			StartID:          weiqitvStartID,
			EndID:            weiqitvEndID,
		}
		wg.Add(1)
		go w.Download(&wg)
	}

	var o *onegreen.Onegreen
	if onegreenEnabled {
		o = &onegreen.Onegreen{
			Semaphore:        *semaphore.NewSemaphore(parallelCount),
			SaveFileEncoding: saveFileEncoding,
			QuitIfExists:     quitIfExists,
		}
		wg.Add(1)
		go o.Download(&wg)
	}

	var t *tom.Tom
	if tomEnabled {
		t = &tom.Tom{
			Semaphore:        *semaphore.NewSemaphore(parallelCount),
			SaveFileEncoding: saveFileEncoding,
			QuitIfExists:     quitIfExists,
		}
		wg.Add(1)
		go t.Download(&wg)
	}
	wg.Wait()

	var downloadCount int32
	if lolEnabled {
		downloadCount += l.DownloadCount
	}
	if sinaEnabled {
		downloadCount += s.DownloadCount
	}
	if tomEnabled {
		downloadCount += t.DownloadCount
	}
	if xgooEnabled {
		downloadCount += x.DownloadCount
	}
	if onegreenEnabled {
		downloadCount += o.DownloadCount
	}
	if weiqitvEnabled {
		downloadCount += w.DownloadCount
	}
	if hoetomEnabled {
		downloadCount += h.DownloadCount
	}
	fmt.Println("total downloaded ", downloadCount, " SGFs")
}
