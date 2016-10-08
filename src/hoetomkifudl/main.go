package main

import (
	"flag"
	"fmt"
	"ic"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"semaphore"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"util"
)

var (
	wg               sync.WaitGroup
	client           *http.Client
	sessionID        string
	userID           string
	password         string
	passwordMd5      string
	saveFileEncoding string
	quit             bool // assume it's false as initial value
	quitIfExists     bool
	latestPageID     int
	earliestPageID   int
	parallelCount    int
	downloadCount    int32
)

func getSessionID() {
	fullURL := fmt.Sprintf("http://www.hoetom.com/servlet/login")
	postBody := fmt.Sprintf("userid=%s&passwd=%s&passwdmd5=%s", userID, password, passwordMd5)
	req, err := http.NewRequest("POST", fullURL, strings.NewReader(postBody))
	if err != nil {
		fmt.Println("Could not parse login request:", err)
		return
	}

	req.Header.Set("Origin", "http://www.hoetom.com")
	req.Header.Set("Referer", "http://www.hoetom.com/index.jsp")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 6.1; WOW64; rv:45.0) Gecko/20100101 Firefox/45.0")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("accept-encoding", `gzip, deflate, sdch`)
	req.Header.Set("accept-language", `en-US,en;q=0.8`)
	req.Header.Set("Cache-Control", "max-age=0")

	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Could not send login request:", err)
		return
	}

	defer resp.Body.Close()

	cookies := resp.Cookies()
	for _, v := range cookies {
		ss := strings.Split(v.String(), ";")
		for _, c := range ss {
			if strings.Index(c, "JSESSIONID") >= 0 {
				sessionID = strings.Split(c, "=")[1]
				return
			}
		}
	}
	fmt.Println("cannot get session id")
}

func downloadKifu(id int, s *semaphore.Semaphore) {
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
	fullURL := fmt.Sprintf("http://www.hoetom.com/chessmanual.jsp?id=%d", id)
	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		fmt.Println("Could not parse kifu request:", err)
		return
	}

	req.Header.Set("Origin", "http://www.hoetom.com")
	req.Header.Set("Referer", "http://www.hoetom.com/matchlatest_pro.jsp")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 6.1; WOW64; rv:45.0) Gecko/20100101 Firefox/45.0")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("accept-encoding", `gzip, deflate, sdch`)
	req.Header.Set("accept-language", `en-US,en;q=0.8`)
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	req.Header.Set("cookie", fmt.Sprintf("JSESSIONID=%s; userid=%s", sessionID, userID))
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
	ss := strings.Split(resp.Header.Get("Content-Disposition"), ";")
	if len(ss) < 2 {
		fmt.Println("cannot get content-disposition")
		retry++
		if retry < 3 {
			time.Sleep(3 * time.Second)
			goto doRequest
		}
		ss = []string{"", fmt.Sprintf(`"id=%d.sgf"`, id)}
	}
	filename := strings.Split(ss[1], "=")[1]
	filename = filename[1 : len(filename)-1]
	filename = ic.ConvertString("gbk", "utf-8", filename)
	dir := fmt.Sprintf("%d", id/1000)
	if !util.Exists(dir) {
		os.MkdirAll(dir, 0777)
	}
	fullPath := fmt.Sprintf("%s/%s", dir, filename)
	if util.Exists(fullPath) {
		if quitIfExists {
			quit = true
		}
		return
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
	fullURL := fmt.Sprintf("http://www.hoetom.com/matchlatest_pro.jsp?pn=%d", page)
	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		fmt.Println("Could not parse page request:", err)
		return
	}

	req.Header.Set("Origin", "http://www.hoetom.com")
	req.Header.Set("Referer", "http://www.hoetom.com/matchlatest_pro.jsp")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 6.1; WOW64; rv:45.0) Gecko/20100101 Firefox/45.0")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("accept-encoding", `gzip, deflate, sdch`)
	req.Header.Set("accept-language", `en-US,en;q=0.8`)
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	req.Header.Set("cookie", fmt.Sprintf("JSESSIONID=%s; userid=%s", sessionID, userID))
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

	regex := regexp.MustCompile(`matchinfor\.jsp\?id=([0-9]+)`)
	ss := regex.FindAllSubmatch(data, -1)
	for _, match := range ss {
		if quit {
			break
		}
		id, err := strconv.Atoi(string(match[1]))
		if err != nil {
			fmt.Printf("converting %s to number failed", string(match[1]))
			continue
		}

		go downloadKifu(id, s)
	}
}

func main() {
	client = &http.Client{
		Timeout: 30 * time.Second,
	}
	flag.StringVar(&saveFileEncoding, "encoding", "gbk", "save SGF file encoding")
	flag.BoolVar(&quitIfExists, "q", false, "quit if the target file exists")
	flag.IntVar(&latestPageID, "l", 1, "the latest page id")
	flag.IntVar(&earliestPageID, "e", 1045, "the earliest page id")
	flag.IntVar(&parallelCount, "p", 20, "the parallel routines count")
	flag.Parse()

	getSessionID()
	fmt.Println("save SGF file encoding", saveFileEncoding)
	fmt.Println("quit if the target file exists", quitIfExists)
	fmt.Println("the latest pid", latestPageID)
	fmt.Println("the earliest pid", earliestPageID)
	fmt.Println("the parallel routines count", parallelCount)
	fmt.Println("session id", sessionID)
	s := semaphore.NewSemaphore(parallelCount)
	for i := latestPageID; i <= earliestPageID && !quit; i++ {
		downloadPage(i, s)
	}

	wg.Wait()
	fmt.Println("Totally downloaded", downloadCount, " SGF files")
}
