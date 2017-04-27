package weiqitv

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"kifudl/ic"
	"kifudl/semaphore"
	"kifudl/util"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"sync"
	"sync/atomic"
	"time"
)

var (
	client *http.Client
)

type WeiqiTV struct {
	sync.WaitGroup
	*semaphore.Semaphore
	SaveFileEncoding string
	quit             bool // assume it's false as initial value
	QuitIfExists     bool
	StartID          int
	EndID            int
	DownloadCount    int32
}

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

func (w *WeiqiTV) downloadKifu(sgf string) {
	w.Add(1)
	w.Acquire()
	defer func() {
		w.Release()
		w.Done()
	}()
	if w.quit {
		return
	}
	retry := 0

	req, err := http.NewRequest("GET", sgf, nil)
	if err != nil {
		log.Println("weiqitv - Could not parse kifu request:", err)
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
		log.Println("weiqitv - Could not send kifu request:", err)
		retry++
		if retry < 3 {
			time.Sleep(3 * time.Second)
			goto doRequest
		}
		return
	}

	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		log.Println("weiqitv - kifu request not 200")
		retry++
		if retry < 3 {
			time.Sleep(3 * time.Second)
			goto doRequest
		}
		return
	}
	kifu, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("weiqitv - cannot read kifu content", err)
		retry++
		if retry < 3 {
			time.Sleep(3 * time.Second)
			goto doRequest
		}
		return
	}

	var kifuInfo KifuInfo
	if err = json.Unmarshal(kifu, &kifuInfo); err != nil {
		log.Println("weiqitv - cannot unmarshal json", err)
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
	fullPath := fmt.Sprintf("weiqitv/%s/%s_%s_%s_vs_%s_%s.sgf",
		util.InsertSlashNth(u.Path[16:], 4), kifuInfo.Name, kifuInfo.B, kifuInfo.LB, kifuInfo.W, kifuInfo.LW)
	if util.Exists(fullPath) {
		if !w.quit && w.QuitIfExists {
			log.Println(fullPath, " exists, just quit")
			w.quit = true
		}
		return
	}

	dir := path.Dir(fullPath)
	if !util.Exists(dir) {
		os.MkdirAll(dir, 0777)
	}
	kifu = []byte(kifuInfo.SGF)
	if w.SaveFileEncoding != "gbk" {
		kifu = ic.Convert("gbk", w.SaveFileEncoding, kifu)
	}
	ioutil.WriteFile(fullPath, kifu, 0644)
	kifu = nil
	atomic.AddInt32(&w.DownloadCount, 1)
}

type Index struct {
	ID string `json:"id"`
}

type Indexes struct {
	Data  []Index `json:"data"`
	Total int     `json:"total"`
}

func (w *WeiqiTV) downloadIndex(id int) (res []string) {
	w.Add(1)
	w.Acquire()
	defer func() {
		w.Release()
		w.Done()
	}()
	if w.quit {
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
		log.Println("weiqitv - Could not parse download index request:", err)
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
		log.Println("weiqitv - Could not send download index request:", err)
		retry++
		if retry < 3 {
			time.Sleep(3 * time.Second)
			goto doRequest
		}
		return
	}

	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		log.Println("weiqitv - index request not 200")
		retry++
		if retry < 3 {
			time.Sleep(3 * time.Second)
			goto doRequest
		}
		return
	}
	index, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("weiqitv - cannot read index content", err, string(index))
		retry++
		if retry < 3 {
			time.Sleep(3 * time.Second)
			goto doRequest
		}
		return
	}

	var indexes Indexes
	if err := json.Unmarshal(index, &indexes); err != nil {
		log.Println("weiqitv - cannot unmarshal indexes", err)
		return
	}

	w.EndID = indexes.Total
	for _, i := range indexes.Data {
		res = append(res, i.ID)
	}

	return res
}

func (w *WeiqiTV) Download(ow *sync.WaitGroup) {
	defer ow.Done()
	client = &http.Client{
		Timeout: 120 * time.Second,
	}

	fmt.Println("weiqitv the latest pid", w.StartID)
	fmt.Println("weiqitv the earliest pid", w.EndID)

	for i := w.StartID; i <= w.EndID && !w.quit; i += step {
		res := w.downloadIndex(i)
		for _, id := range res {
			sgf := fmt.Sprintf("http://yi.weiqitv.com/pub/kifureview/%s", id)
			go w.downloadKifu(sgf)
		}
	}

	w.Wait()
	fmt.Println("downloaded", w.DownloadCount, " SGF files from WeiqiTV")
}
