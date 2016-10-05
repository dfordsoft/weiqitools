package main

import (
	"flag"
	"fmt"
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
	wg                sync.WaitGroup
	client            *http.Client
	sessionID         string
	userID            string
	password          string
	passwordMd5       string
	quit              bool // assume it's false as initial value
	quitIfExists      bool
	runIconvAfterSave bool
	latestPageID      int
	earliestPageID    int
	parallelCount     int
	downloadCount     int32
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

	fullURL := fmt.Sprintf("http://www.hoetom.com/chessmanual.jsp?id=%d", id)
	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		fmt.Println("Could not parse login request:", err)
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

	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Could not send login request:", err)
		return
	}

	defer resp.Body.Close()
	kifu, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("cannot read page content", err)
		return
	}
	ss := strings.Split(resp.Header.Get("Content-Disposition"), ";")[1]
	filename := strings.Split(ss, "=")[1]
	// fmt.Println(filename, []byte(filename))
	// filename, _ = url.QueryUnescape(filename)
	// fmt.Println(filename, []byte(filename))
	// filename = url.QueryEscape(filename)
	// fmt.Println(filename, []byte(filename))
	// r := bytes.NewReader([]byte(filename))
	// d, err := charset.NewReader(r, "gbk")
	// content, err := ioutil.ReadAll(d)
	// fmt.Println(string(content), content)

	dir := fmt.Sprintf("%d", id/1000)
	if !util.Exists(dir) {
		os.MkdirAll(dir, 0777)
	}
	// r := bytes.NewReader([]byte(kifu))
	// d, err := charset.NewReader(r, "gb18030")
	// content, err := ioutil.ReadAll(d)
	ioutil.WriteFile(fmt.Sprintf("%s/%s", dir, filename), kifu, 0644)
	atomic.AddInt32(&downloadCount, 1)
}

func downloadPage(page int, s *semaphore.Semaphore) {
	wg.Add(1)
	defer func() {
		wg.Done()
	}()

	fullURL := fmt.Sprintf("http://www.hoetom.com/matchlatest_pro.jsp?pn=%d", page)
	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		fmt.Println("Could not parse login request:", err)
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

	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Could not send login request:", err)
		return
	}

	defer resp.Body.Close()
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("cannot read page content", err)
		return
	}

	regex := regexp.MustCompile(`matchinfor\.jsp\?id=([0-9]+)`)
	ss := regex.FindAllSubmatch(data, -1)
	for _, match := range ss {
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

	flag.BoolVar(&runIconvAfterSave, "iconv", false, "run iconv after file saved")
	flag.BoolVar(&quitIfExists, "q", true, "quit if the target file exists")
	flag.IntVar(&latestPageID, "l", 1, "the latest page id")
	flag.IntVar(&earliestPageID, "e", 1045, "the earliest page id")
	flag.IntVar(&parallelCount, "p", 20, "the parallel routines count")
	flag.StringVar(&userID, "user", "missdeer", "login as user")
	flag.StringVar(&password, "passwd", "", "user password")
	flag.StringVar(&passwordMd5, "md5", "11FA17CDAB3FF2C39BB5E781BEAE646D", "user password md5")
	flag.Parse()

	getSessionID()
	fmt.Println("run iconv after file saved", runIconvAfterSave)
	fmt.Println("quit if the target file exists", quitIfExists)
	fmt.Println("the latest pid", latestPageID)
	fmt.Println("the earliest pid", earliestPageID)
	fmt.Println("the parallel routines count", parallelCount)
	fmt.Println("user", userID)
	fmt.Println("user password", password)
	fmt.Println("user password md5", passwordMd5)
	fmt.Println("session id", sessionID)
	s := semaphore.NewSemaphore(parallelCount)
	for i := latestPageID; i <= earliestPageID && !quit; i++ {
		downloadPage(i, s)
	}

	wg.Wait()
	fmt.Println("Totally downloaded", downloadCount, " SGF files")
}
