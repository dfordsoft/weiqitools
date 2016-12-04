package weiqitv

import (
	"encoding/json"
	"flag"
	"fmt"
	"ic"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"semaphore"
	"sync"
	"sync/atomic"
	"time"
	"util"
)

var (
	wg               sync.WaitGroup
	client           *http.Client
	saveFileEncoding string
	quit             bool // assume it's false as initial value
	quitIfExists     bool
	startID          int
	endID            int
	parallelCount    int
	downloadCount    int32
)

const (
	step = 10 // weiqitv.com always returns 10 records no matter what value is set
)

type KifuInfo struct {
	ID     string `json:"id"`
	LB     string `json:"lb"`
	LW     string `json:"lw"`
	B      string `json:"b"`
	W      string `json:"w"`
	SGF    string `json:"sgf"`
	Name   string `json:"name"`
	Result string `json:"result"`
}

func downloadKifu(sgf string, s *semaphore.Semaphore) {
	wg.Add(1)
	s.Acquire()
	defer func() {
		s.Release()
		wg.Done()
	}()
	if quit {
		return
	}
	retry := 0

	req, err := http.NewRequest("GET", sgf, nil)
	if err != nil {
		fmt.Println("Could not parse kifu request:", err)
		return
	}

	req.Header.Set("Referer", "http://yi.weiqitv.com/pub/kifu")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 6.1; WOW64; rv:45.0) Gecko/20100101 Firefox/45.0")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("accept-language", `en-US,en;q=0.8`)
	req.Header.Set("Upgrade-Insecure-Requests", "1")
doRequest:
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Could not send kifu request:", err)
		retry++
		if retry < 3 {
			time.Sleep(3 * time.Second)
			goto doRequest
		}
		return
	}

	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		fmt.Println("kifu request not 200")
		retry++
		if retry < 3 {
			time.Sleep(3 * time.Second)
			goto doRequest
		}
		return
	}
	kifu, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("cannot read kifu content", err)
		retry++
		if retry < 3 {
			time.Sleep(3 * time.Second)
			goto doRequest
		}
		return
	}

	var kifuInfo KifuInfo
	if err := json.Unmarshal(kifu, &kifuInfo); err != nil {
		fmt.Println("cannot unmarshal json", err)
		retry++
		if retry < 3 {
			time.Sleep(3 * time.Second)
			goto doRequest
		}
		return
	}

	u, err := url.Parse(sgf)
	if err != nil {
		log.Fatal(err)
	}
	fullPath := fmt.Sprintf("%s_%s_%s_%s_vs_%s_%s.sgf",
		u.Path[1:], kifuInfo.Name, kifuInfo.B, kifuInfo.LB, kifuInfo.W, kifuInfo.LW)
	if util.Exists(fullPath) {
		if quitIfExists {
			quit = true
		}
		return
	}

	dir := path.Dir(fullPath)
	if !util.Exists(dir) {
		os.MkdirAll(dir, 0777)
	}
	kifu = []byte(kifuInfo.SGF)
	if saveFileEncoding != "gbk" {
		kifu = ic.Convert("gbk", saveFileEncoding, kifu)
	}
	ioutil.WriteFile(fullPath, kifu, 0644)
	kifu = nil
	atomic.AddInt32(&downloadCount, 1)
}

type Index struct {
	ID string `json:"id"`
}

type Indexes struct {
	Data  []Index `json:"data"`
	Total int     `json:"total"`
}

func downloadIndex(id int, s *semaphore.Semaphore) (res []string) {
	wg.Add(1)
	s.Acquire()
	defer func() {
		s.Release()
		wg.Done()
	}()
	if quit {
		return
	}
	retry := 0
	getValues := url.Values{
		"start":    {fmt.Sprintf("%d", id)},
		"len":      {fmt.Sprintf("%d", step)},
		"kifuTp":   {"全部"},
		"gameSort": {"false"},
	}

	req, err := http.NewRequest("GET", `http://yi.weiqitv.com/pub/kifu?`+getValues.Encode(), nil)
	if err != nil {
		fmt.Println("Could not parse download index request:", err)
		return
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 6.1; WOW64; rv:45.0) Gecko/20100101 Firefox/45.0")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("accept-language", `en-US,en;q=0.8`)
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	req.Header.Set("Cache-Control", "max-age=0")
doRequest:
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Could not send download index request:", err)
		retry++
		if retry < 3 {
			time.Sleep(3 * time.Second)
			goto doRequest
		}
		return
	}

	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		fmt.Println("index request not 200")
		retry++
		if retry < 3 {
			time.Sleep(3 * time.Second)
			goto doRequest
		}
		return
	}
	index, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("cannot read index content", err, string(index))
		retry++
		if retry < 3 {
			time.Sleep(3 * time.Second)
			goto doRequest
		}
		return
	}

	var indexes Indexes
	if err := json.Unmarshal(index, &indexes); err != nil {
		fmt.Println("cannot unmarshal indexes", err)
		return
	}

	endID = indexes.Total
	for _, i := range indexes.Data {
		res = append(res, i.ID)
	}

	return res
}

func main() {
	client = &http.Client{
		Timeout: 120 * time.Second,
	}
	flag.StringVar(&saveFileEncoding, "encoding", "gbk", "save SGF file encoding")
	flag.BoolVar(&quitIfExists, "q", true, "quit if the target file exists")
	flag.IntVar(&startID, "l", 0, "the start id")
	flag.IntVar(&endID, "e", 77281, "the end id")
	flag.IntVar(&parallelCount, "p", 20, "the parallel routines count")
	flag.Parse()

	fmt.Println("save SGF file encoding", saveFileEncoding)
	fmt.Println("quit if the target file exists", quitIfExists)
	fmt.Println("the latest pid", startID)
	fmt.Println("the earliest pid", endID)

	s := semaphore.NewSemaphore(parallelCount)
	for i := startID; i <= endID && !quit; i += step {
		res := downloadIndex(i, s)
		for _, id := range res {
			sgf := fmt.Sprintf("http://yi.weiqitv.com/pub/kifureview/%s", id)
			go downloadKifu(sgf, s)
		}
	}

	wg.Wait()
	fmt.Println("Totally downloaded", downloadCount, " SGF files")
}
