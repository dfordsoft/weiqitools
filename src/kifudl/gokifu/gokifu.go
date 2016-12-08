package gokifu

import (
	"bytes"
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
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var (
	client *http.Client
)

type GoKifu struct {
	sync.WaitGroup
	semaphore.Semaphore
	SaveFileEncoding string
	quit             bool // assume it's false as initial value
	QuitIfExists     bool
	LatestPageID     int
	EarliestPageID   int
	DownloadCount    int32
}

func (g *GoKifu) downloadKifu(sgf string) {
	g.Add(1)
	g.Acquire()
	defer func() {
		g.Release()
		g.Done()
	}()
	if g.quit {
		return
	}
	retry := 0

	req, err := http.NewRequest("GET", sgf, nil)
	if err != nil {
		log.Println("Could not parse kifu request:", err)
		return
	}

	req.Header.Set("Referer", "http://gokifu.com/?p=123")
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

	u, err := url.Parse(url.QueryEscape(sgf))
	if err != nil {
		log.Fatal(err)
	}
	filename := u.Path[3:]
	ff := strings.Split(filename, "-")
	date := util.InsertSlashNth(ff[2], 4)
	filename = strings.Join(ff[3:], "-")
	fullPath := fmt.Sprintf("gokifu/%s/%s", date, filename)
	if util.Exists(fullPath) {
		if g.QuitIfExists {
			log.Println(fullPath, " exists, just quit")
			g.quit = true
		}
		return
	}

	dir := path.Dir(fullPath)
	if !util.Exists(dir) {
		os.MkdirAll(dir, 0777)
	}
	if g.SaveFileEncoding != "utf-8" {
		kifu = ic.Convert("utf-8", g.SaveFileEncoding, kifu)
		kifu = bytes.Replace(kifu, []byte("UTF-8"), []byte(g.SaveFileEncoding), 1)
	}
	ioutil.WriteFile(fullPath, kifu, 0644)
	kifu = nil
	atomic.AddInt32(&g.DownloadCount, 1)
}

func (g *GoKifu) downloadPage(page int) {
	g.Add(1)
	g.Acquire()
	defer func() {
		g.Release()
		g.Done()
	}()
	retry := 0
	fullURL := fmt.Sprintf("http://gokifu.com/zh/index.php?p=%d", page)
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

	regex := regexp.MustCompile(`http:\/\/gokifu\.com\/zh\/f\/[^"]+`)
	ss := regex.FindAll(data, -1)
	dl := make(map[string]bool, len(ss))
	for _, match := range ss {
		if g.quit {
			break
		}
		sgf := string(match)
		if _, ok := dl[sgf]; ok {
			continue
		}
		dl[sgf] = true
		go g.downloadKifu(sgf)
	}

	dl = nil
}

func (g *GoKifu) Download(w *sync.WaitGroup) {
	defer w.Done()
	client = &http.Client{
		Timeout: 30 * time.Second,
	}

	fmt.Println("gokifu the latest pid", g.LatestPageID)
	fmt.Println("gokifu the earliest pid", g.EarliestPageID)

	for i := g.LatestPageID; i <= g.EarliestPageID && !g.quit; i++ {
		g.downloadPage(i)
	}

	g.Wait()
	fmt.Println("downloaded", g.DownloadCount, " SGF files from GoKifu")
}
