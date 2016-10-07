package main

import (
	"flag"
	"fmt"
	"ic"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"regexp"
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
	latestPageID     int
	earliestPageID   int
	parallelCount    int
	downloadCount    int32
)

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

	req.Header.Set("Referer", "http://duiyi.sina.com.cn/gibo/new_gibo.asp?cur_page=689")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 6.1; WOW64; rv:45.0) Gecko/20100101 Firefox/45.0")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("accept-encoding", `gzip, deflate, sdch`)
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

	u, err := url.Parse(sgf)
	if err != nil {
		log.Fatal(err)
	}
	fullPath := u.Path[1:]
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
	if saveFileEncoding != "gbk" {
		kifu = ic.Convert("gbk", saveFileEncoding, kifu)
	}
	ioutil.WriteFile(fullPath, kifu, 0644)
	kifu = nil
	atomic.AddInt32(&downloadCount, 1)
}

func downloadPage(page int, s *semaphore.Semaphore) {
	wg.Add(1)
	s.Acquire()
	defer func() {
		s.Release()
		wg.Done()
	}()
	retry := 0
	fullURL := fmt.Sprintf("http://duiyi.sina.com.cn/gibo/new_gibo.asp?cur_page=%d", page)
	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		fmt.Println("Could not parse page request:", err)
		return
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 6.1; WOW64; rv:45.0) Gecko/20100101 Firefox/45.0")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("accept-encoding", `gzip, deflate, sdch`)
	req.Header.Set("accept-language", `en-US,en;q=0.8`)
doPageRequest:
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Could not send page request:", err)
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
		fmt.Println("cannot read page content", err)
		retry++
		if retry < 3 {
			time.Sleep(3 * time.Second)
			goto doPageRequest
		}
		return
	}

	regex := regexp.MustCompile(`JavaScript:gibo_load\('(http:\/\/duiyi\.sina\.com\.cn\/cgibo\/[0-9]+\/[0-9a-zA-Z\-]+\.sgf)'\)`)
	ss := regex.FindAllSubmatch(data, -1)
	dl := make(map[string]bool, len(ss))
	for _, match := range ss {
		if quit {
			break
		}
		sgf := string(match[1])
		if _, ok := dl[sgf]; ok {
			continue
		}
		dl[sgf] = true
		go downloadKifu(sgf, s)
	}

	regex = regexp.MustCompile(`JavaScript:gibo_load\('(http:\/\/duiyi\.sina\.com\.cn\/cgibo\/[0-9a-zA-Z\-]+\.sgf)'\)`)
	ss = regex.FindAllSubmatch(data, -1)
	dl = make(map[string]bool, len(ss))
	for _, match := range ss {
		if quit {
			break
		}
		sgf := string(match[1])
		if _, ok := dl[sgf]; ok {
			continue
		}
		dl[sgf] = true
		go downloadKifu(sgf, s)
	}
	dl = nil
}

func main() {
	client = &http.Client{
		Timeout: 30 * time.Second,
	}
	flag.StringVar(&saveFileEncoding, "encoding", "utf-8", "save SGF file encoding")
	flag.BoolVar(&quitIfExists, "q", false, "quit if the target file exists")
	flag.IntVar(&latestPageID, "l", 0, "the latest page id")
	flag.IntVar(&earliestPageID, "e", 689, "the earliest page id")
	flag.IntVar(&parallelCount, "p", 20, "the parallel routines count")
	flag.Parse()

	fmt.Println("save SGF file encoding", saveFileEncoding)
	fmt.Println("quit if the target file exists", quitIfExists)
	fmt.Println("the latest pid", latestPageID)
	fmt.Println("the earliest pid", earliestPageID)

	s := semaphore.NewSemaphore(parallelCount)
	for i := latestPageID; i <= earliestPageID && !quit; i++ {
		downloadPage(i, s)
	}

	wg.Wait()
	fmt.Println("Totally downloaded", downloadCount, " SGF files")
}
