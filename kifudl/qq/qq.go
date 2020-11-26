package qq

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"github.com/missdeer/weiqitools/kifudl/util"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"sync/atomic"
	"time"

	"github.com/missdeer/golib/ic"
	"github.com/missdeer/golib/semaphore"
	"github.com/missdeer/golib/fsutil"
)

var (
	client *http.Client
)

type QQ struct {
	sync.WaitGroup
	*semaphore.Semaphore
	SaveFileEncoding string
	quit             bool // assume it's false as initial value
	QuitIfExists     bool
	LatestPageID     int
	EarliestPageID   int
	DownloadCount    int32
}

func (q *QQ) downloadKifu(sgf string) {
	q.Add(1)
	q.Acquire()
	defer func() {
		q.Release()
		q.Done()
	}()
	if q.quit {
		return
	}
	retry := 0

	req, err := http.NewRequest("GET", sgf, nil)
	if err != nil {
		log.Println("qq - Could not parse kifu request:", err)
		return
	}

	req.Header.Set("Referer", "http://qipu.qq.org/indeq.php?page=1")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 6.1; WOW64; rv:45.0) Gecko/20100101 Firefox/45.0")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("accept-language", `en-US,en;q=0.8`)
	req.Header.Set("Upgrade-Insecure-Requests", "1")
doRequest:
	resp, err := client.Do(req)
	if err != nil {
		log.Println("qq - Could not send kifu request:", err)
		retry++
		if retry < 3 {
			time.Sleep(3 * time.Second)
			goto doRequest
		}
		return
	}

	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		log.Println("qq - kifu request not 200")
		retry++
		if retry < 3 {
			time.Sleep(3 * time.Second)
			goto doRequest
		}
		return
	}
	kifu, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("qq - cannot read kifu content", err)
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
	base := filepath.Base(u.Path)
	fullPath := filepath.Join("qq", base[:4], base[4:len(base)-8]+".sgf")
	if util.Exists(fullPath) {
		if !q.quit && q.QuitIfExists {
			log.Println(fullPath, " exists, just quit")
			q.quit = true
		}
		return
	}

	dir := filepath.Dir(fullPath)
	if b, e := fsutil.FileExists(dir); e != nil || !b {
		os.MkdirAll(dir, 0777)
	}
	if q.SaveFileEncoding != "gbk" {
		kifu = ic.Convert("gbk", q.SaveFileEncoding, kifu)
	}

	// extract kifu content
	leadingStr := []byte(`id="player-container">`)
	index := bytes.Index(kifu, leadingStr)
	if index < 0 {
		return
	}
	kifu = kifu[index+len(leadingStr):]
	endingStr := []byte(`</div>`)
	index = bytes.Index(kifu, endingStr)
	if index < 0 {
		return
	}
	kifu = kifu[:index]

	err = ioutil.WriteFile(fullPath, kifu, 0644)
	if err != nil {
		log.Println(err)
	}
	kifu = nil
	atomic.AddInt32(&q.DownloadCount, 1)
}

func (q *QQ) downloadPage(page int) {
	q.Add(1)
	q.Acquire()
	defer func() {
		q.Release()
		q.Done()
	}()
	retry := 0
	fullURL := fmt.Sprintf("http://weiqi.qq.com/qipu/index/p/%d.html", page)
	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		log.Println("qq - Could not parse page request:", err)
		return
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 6.1; WOW64; rv:45.0) Gecko/20100101 Firefox/45.0")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("accept-language", `en-US,en;q=0.8`)
doPageRequest:
	resp, err := client.Do(req)
	if err != nil {
		log.Println("qq - Could not send page request:", err)
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
		log.Println("qq - cannot read page content", err)
		retry++
		if retry < 3 {
			time.Sleep(3 * time.Second)
			goto doPageRequest
		}
		return
	}

	regex := regexp.MustCompile(`href="(\/qipu\/newlist\/id\/[0-9]+\.html)"`)
	ss := regex.FindAllSubmatch(data, -1)
	for _, match := range ss {
		if q.quit {
			break
		}
		sgfURL := "http://weiqi.qq.com" + string(match[1])
		go q.downloadKifu(sgfURL)
	}
}

func (q *QQ) Download(w *sync.WaitGroup) {
	defer w.Done()
	client = &http.Client{
		Timeout: 30 * time.Second,
	}

	fmt.Println("qq the latest page id", q.LatestPageID)
	fmt.Println("qq the earliest page id", q.EarliestPageID)

	for i := q.LatestPageID; i <= q.EarliestPageID && !q.quit; i++ {
		q.downloadPage(i)
	}

	q.Wait()
	fmt.Println("downloaded", q.DownloadCount, " SGF files from QQ")
}
