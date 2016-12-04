package xgoo

import (
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
	"regexp"
	"sync"
	"sync/atomic"
	"time"
)

var (
	client *http.Client
)

type Xgoo struct {
	sync.WaitGroup
	Sem              *semaphore.Semaphore
	SaveFileEncoding string
	quit             bool // assume it's false as initial value
	QuitIfExists     bool
	LatestPageID     int
	EarliestPageID   int
	DownloadCount    int32
}

func (x *Xgoo) downloadKifu(sgf string) {
	x.Add(1)
	x.Sem.Acquire()
	defer func() {
		x.Sem.Release()
		x.Done()
	}()
	if x.quit {
		return
	}
	retry := 0

	req, err := http.NewRequest("GET", sgf, nil)
	if err != nil {
		log.Println("Could not parse kifu request:", err)
		return
	}

	req.Header.Set("Referer", "http://qipu.xgoo.org/index.php?page=1")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 6.1; WOW64; rv:45.0) Gecko/20100101 Firefox/45.0")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("accept-language", `en-US,en;q=0.8`)
	req.Header.Set("Upgrade-Insecure-Requests", "1")
doRequest:
	resp, err := client.Do(req)
	if err != nil {
		log.Println("Could not send kifu request:", err)
		retry++
		if retry < 3 {
			time.Sleep(3 * time.Second)
			goto doRequest
		}
		return
	}

	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		log.Println("kifu request not 200")
		retry++
		if retry < 3 {
			time.Sleep(3 * time.Second)
			goto doRequest
		}
		return
	}
	kifu, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("cannot read kifu content", err)
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
	fullPath := "xgoo/" + u.Path[1:]
	if util.Exists(fullPath) {
		if x.QuitIfExists {
			x.quit = true
		}
		return
	}

	dir := path.Dir(fullPath)
	if !util.Exists(dir) {
		os.MkdirAll(dir, 0777)
	}
	if x.SaveFileEncoding != "gbk" {
		kifu = ic.Convert("gbk", x.SaveFileEncoding, kifu)
	}
	ioutil.WriteFile(fullPath, kifu, 0644)
	kifu = nil
	atomic.AddInt32(&x.DownloadCount, 1)
}

func (x *Xgoo) downloadPage(page int) {
	x.Add(1)
	x.Sem.Acquire()
	defer func() {
		x.Sem.Release()
		x.Done()
	}()
	retry := 0
	fullURL := fmt.Sprintf("http://qipu.xgoo.org/index.php?page=%d", page)
	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		log.Println("Could not parse page request:", err)
		return
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 6.1; WOW64; rv:45.0) Gecko/20100101 Firefox/45.0")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("accept-language", `en-US,en;q=0.8`)
doPageRequest:
	resp, err := client.Do(req)
	if err != nil {
		log.Println("Could not send page request:", err)
		retry++
		if retry < 3 {
			time.Sleep(3 * time.Second)
			goto doPageRequest
		}
		return
	}

	defer resp.Body.Close()
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("cannot read page content", err)
		retry++
		if retry < 3 {
			time.Sleep(3 * time.Second)
			goto doPageRequest
		}
		return
	}

	regex := regexp.MustCompile(`href=(http:\/\/www\.xgoo\.org\/qipu\/[0-9a-zA-Z\-\_\/]+\.sgf)`)
	ss := regex.FindAllSubmatch(data, -1)
	for _, match := range ss {
		if x.quit {
			break
		}
		sgf := string(match[1])
		go x.downloadKifu(sgf)
	}
}

func (x *Xgoo) Download(w *sync.WaitGroup) {
	w.Add(1)
	defer w.Done()
	client = &http.Client{
		Timeout: 30 * time.Second,
	}

	fmt.Println("the latest pid", x.LatestPageID)
	fmt.Println("the earliest pid", x.EarliestPageID)

	for i := x.LatestPageID; i <= x.EarliestPageID && !x.quit; i++ {
		x.downloadPage(i)
	}

	x.Wait()
	fmt.Println("Totally downloaded", x.DownloadCount, " SGF files")
}
