package main

import (
	"fmt"
	"github.com/missdeer/weiqitools/kifudl/gokifu"
	"github.com/missdeer/weiqitools/kifudl/hotongo"
	"github.com/missdeer/weiqitools/kifudl/lol"
	"github.com/missdeer/weiqitools/kifudl/onegreen"
	"github.com/missdeer/weiqitools/kifudl/qq"
	"github.com/missdeer/weiqitools/kifudl/sina"
	"github.com/missdeer/weiqitools/kifudl/xgoo"
	"sync"

	"github.com/missdeer/golib/semaphore"

	flag "github.com/ogier/pflag"
)

func main() {
	var quitIfExists bool
	var saveFileEncoding string
	var parallelCount int
	var lolEnabled, xgooEnabled, sinaEnabled, onegreenEnabled, hotongoEnabled, gokifuEnabled, qqEnabled bool
	flag.StringVar(&saveFileEncoding, "encoding", "gbk", "save SGF file encoding")
	flag.BoolVar(&quitIfExists, "q", true, "quit if the target file exists")
	flag.IntVar(&parallelCount, "p", 20, "the parallel routines count")
	flag.BoolVar(&lolEnabled, "lol-enabled", true, "fetch kifu from lol")
	flag.BoolVar(&xgooEnabled, "xgoo-enabled", true, "fetch kifu from xgoo")
	flag.BoolVar(&sinaEnabled, "sina-enabled", true, "fetch kifu from sina")
	flag.BoolVar(&onegreenEnabled, "onegreen-enabled", true, "fetch kifu from onegreen")
	flag.BoolVar(&hotongoEnabled, "hotongo-enabled", true, "fetch kifu from hotongo")
	flag.BoolVar(&gokifuEnabled, "gokifu-enabled", true, "fetch kifu from gokifu")
	flag.BoolVar(&qqEnabled, "qq-enabled", true, "fetch kifu from qq")

	var hotongoLatestPageID, hotongoEarliestPageID int
	flag.IntVar(&hotongoLatestPageID, "hotongo-latest-page-id", 1, "the latest page id of hotongo")
	flag.IntVar(&hotongoEarliestPageID, "hotongo-earliest-page-id", 1433, "the earliest page id of hotongo")

	var lolLatestID, lolEarliestID int
	flag.IntVar(&lolLatestID, "lol-latest-id", 0, "the latest pid of 101weiqi")
	flag.IntVar(&lolEarliestID, "lol-earliest-id", 1, "the earliest pid of 101weiqi")

	var sinaLatestPageID, sinaEarliestPageID int
	flag.IntVar(&sinaLatestPageID, "sina-latest-page-id", 0, "the latest page id of sina")
	flag.IntVar(&sinaEarliestPageID, "sina-earliest-page-id", 689, "the earliest page id of sina")

	var gokifuLatestPageID, gokifuEarliestPageID int
	flag.IntVar(&gokifuLatestPageID, "gokifu-latest-page-id", 1, "the latest page id of gokifu")
	flag.IntVar(&gokifuEarliestPageID, "gokifu-earliest-page-id", 1818, "the earliest page id of gokifu")

	var xgooLatestPageID, xgooEarliestPageID int
	flag.IntVar(&xgooLatestPageID, "xgoo-latest-page-id", 1, "the latest page id of xgoo")
	flag.IntVar(&xgooEarliestPageID, "xgoo-earliest-page-id", 1968, "the earliest page id of xgoo")

	var qqLatestPageID, qqEarliestPageID int
	flag.IntVar(&qqLatestPageID, "qq-latest-page-id", 1, "the latest page id of qq")
	flag.IntVar(&qqEarliestPageID, "qq-earliest-page-id", 34, "the earliest page id of qq")
	flag.Parse()

	fmt.Println("Kifu downloader (c) 2016 - 2019 https://minidump.info & me@minidump.info. All right reserved.")
	fmt.Println("save SGF file encoding", saveFileEncoding)
	fmt.Println("quit if the target file exists", quitIfExists)
	fmt.Println("the parallel routines count", parallelCount)

	var wg sync.WaitGroup
	sem := semaphore.New(parallelCount)
	var h *hotongo.Hotongo
	if hotongoEnabled {
		h = &hotongo.Hotongo{
			Semaphore:        sem,
			SaveFileEncoding: saveFileEncoding,
			QuitIfExists:     quitIfExists,
			LatestPageID:     hotongoLatestPageID,
			EarliestPageID:   hotongoEarliestPageID,
		}
		wg.Add(1)
		go h.Download(&wg)
	}

	var l *lol.Lol
	if lolEnabled {
		l = &lol.Lol{
			Semaphore:        sem,
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
		s = &sina.Sina{
			Semaphore:        sem,
			SaveFileEncoding: saveFileEncoding,
			QuitIfExists:     quitIfExists,
			LatestPageID:     sinaLatestPageID,
			EarliestPageID:   sinaEarliestPageID,
		}
		wg.Add(1)
		go s.Download(&wg)
	}

	var g *gokifu.GoKifu
	if gokifuEnabled {
		g = &gokifu.GoKifu{
			Semaphore:        sem,
			SaveFileEncoding: saveFileEncoding,
			QuitIfExists:     quitIfExists,
			LatestPageID:     gokifuLatestPageID,
			EarliestPageID:   gokifuEarliestPageID,
		}
		wg.Add(1)
		go g.Download(&wg)
	}

	var x *xgoo.Xgoo
	if xgooEnabled {
		x = &xgoo.Xgoo{
			Semaphore:        sem,
			SaveFileEncoding: saveFileEncoding,
			QuitIfExists:     quitIfExists,
			LatestPageID:     xgooLatestPageID,
			EarliestPageID:   xgooEarliestPageID,
		}
		wg.Add(1)
		go x.Download(&wg)
	}

	var o *onegreen.Onegreen
	if onegreenEnabled {
		o = &onegreen.Onegreen{
			Semaphore:        sem,
			SaveFileEncoding: saveFileEncoding,
			QuitIfExists:     quitIfExists,
		}
		wg.Add(1)
		go o.Download(&wg)
	}

	var q *qq.QQ
	if qqEnabled {
		q = &qq.QQ{
			Semaphore:        sem,
			SaveFileEncoding: saveFileEncoding,
			QuitIfExists:     quitIfExists,
			LatestPageID:     qqLatestPageID,
			EarliestPageID:   qqEarliestPageID,
		}
		wg.Add(1)
		go q.Download(&wg)
	}
	wg.Wait()
	var downloadCount int32
	if lolEnabled {
		downloadCount += l.DownloadCount
	}
	if sinaEnabled {
		downloadCount += s.DownloadCount
	}
	if xgooEnabled {
		downloadCount += x.DownloadCount
	}
	if onegreenEnabled {
		downloadCount += o.DownloadCount
	}
	if hotongoEnabled {
		downloadCount += h.DownloadCount
	}
	if gokifuEnabled {
		downloadCount += g.DownloadCount
	}
	if qqEnabled {
		downloadCount += q.DownloadCount
	}
	fmt.Println("total downloaded ", downloadCount, " SGFs")
	var c byte
	fmt.Scanln(&c)
}
