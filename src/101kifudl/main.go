package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"ic"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"semaphore"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"util"
)

var (
	wg                  sync.WaitGroup
	client              *http.Client
	csrfmiddlewaretoken string
	csrftoken           string
	quit                bool // assume it's false as initial value
	quitIfExists        bool
	saveFileEncoding    string
	latestID            int
	earliestID          int
	parallelCount       int
	downloadCount       int32
)

func getContent(path string) []byte {
	fullURL := fmt.Sprintf("http://101weiqi.com%s", path)
	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		fmt.Println("Could not parse get kifu request:", err)
		return []byte("")
	}

	req.Header.Set("Referer", "http://101weiqi.com")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 6.1; WOW64; rv:45.0) Gecko/20100101 Firefox/45.0")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("accept-encoding", `gzip, deflate, sdch`)
	req.Header.Set("accept-language", `en-US,en;q=0.8`)
	req.Header.Set("cookie", fmt.Sprintf("csrftoken=%s", csrftoken))

	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Could not send get kifu request:", err)
		return []byte("")
	}

	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		fmt.Println("kifu request not 200")
		return []byte("")
	}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("can't read kifu content", err)
		return []byte("")
	}
	if saveFileEncoding != "utf-8" {
		data = bytes.Replace(data, []byte("utf-8"), []byte(saveFileEncoding), 1)
	}
	return data[3:] // remove BOM
}

type KifuPathResponse struct {
	Result int    `json:"result"`
	PURL   string `json:"purl"`
}

func getPath(index int) string {
	// login
	data := fmt.Sprintf(`pid=%d&csrfmiddlewaretoken=%s`, index, csrfmiddlewaretoken)
	req, err := http.NewRequest("POST", "http://www.101weiqi.com/chessbook/download_sgf/", bytes.NewBufferString(data))
	if err != nil {
		fmt.Println("Could not parse get kifu path request:", err)
		return ""
	}

	req.Header.Set("Referer", fmt.Sprintf("http://www.101weiqi.com/chessbook/chess/%d/", index))
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 6.1; WOW64; rv:45.0) Gecko/20100101 Firefox/45.0")
	req.Header.Set("Accept", "application/json, text/javascript, */*; q=0.01")
	req.Header.Set("X-Requested-With", `XMLHttpRequest`)
	req.Header.Set("Content-Type", `application/x-www-form-urlencoded; charset=UTF-8`)
	req.Header.Set("cookie", fmt.Sprintf("csrftoken=%s", csrftoken))

	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Could not send get kifu path request:", err)
		return ""
	}

	defer resp.Body.Close()
	d, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("can't read kifu path")
		return ""
	}
	var m KifuPathResponse
	err = json.Unmarshal(d, &m)
	if err != nil {
		fmt.Println("can't unmarshal json", string(d), err)
		return ""
	}
	return m.PURL
}

func download(index int, s *semaphore.Semaphore) {
	wg.Add(1)
	defer func() {
		s.Release()
		wg.Done()
	}()
	tryGettingPath := 1
startGettingPath:
	p := getPath(index)
	if len(p) > 0 {
		if quit {
			return
		}
		if strings.Index(p, `小围`) > 0 || strings.Index(p, `大围`) > 0 || strings.Index(p, `老围`) > 0 {
			return
		}
		fullPath := p[1:]
		if !util.Exists(fullPath) {
			tryGettingContent := 1
		startGettingContent:
			if quit {
				return
			}
			kifu := getContent(p)
			if len(kifu) > 0 {
				if bytes.Index(kifu, []byte("EV[]")) > 0 {
					fmt.Println("empty ev node for", fullPath)
					return
				}
				dir, _ := filepath.Split(fullPath)
				if !util.Exists(dir) {
					os.MkdirAll(dir, 0777)
				}
				if saveFileEncoding != "utf-8" {
					kifu = ic.Convert("utf-8", saveFileEncoding, kifu)
				}
				// save to file
				ioutil.WriteFile(fullPath, kifu, 0644)
				kifu = nil
				atomic.AddInt32(&downloadCount, 1)
			} else {
				tryGettingContent++
				if tryGettingContent < 10 {
					time.Sleep(3 * time.Second)
					goto startGettingContent
				}
			}
		} else {
			if quitIfExists {
				fmt.Println(fullPath, "exists, quit now")
				quit = true
			}
		}
	} else {
		tryGettingPath++
		if tryGettingPath < 10 {
			time.Sleep(3 * time.Second)
			goto startGettingPath
		}
	}
}

func getCSRF() {
	fullURL := fmt.Sprintf("http://101weiqi.com/chessbook/chess/%d", latestID)
	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		fmt.Println("Could not parse get latest ID page request:", err)
		return
	}

	req.Header.Set("Referer", "http://101weiqi.com")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 6.1; WOW64; rv:45.0) Gecko/20100101 Firefox/45.0")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("accept-encoding", `gzip, deflate, sdch`)
	req.Header.Set("accept-language", `en-US,en;q=0.8`)

	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Could not send get latest ID page request:", err)
		return
	}

	defer resp.Body.Close()
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("could not read latest ID page")
		return
	}

	startPos := bytes.Index(data, []byte("csrfmiddlewaretoken"))
	if startPos < 0 {
		fmt.Println("could not find csrfmiddlewaretoken")
		return
	}
	csrfmwtokenPos := bytes.Index(data[startPos:], []byte("value='"))

	csrfmiddlewaretoken = string(data[startPos+csrfmwtokenPos+7 : startPos+csrfmwtokenPos+7+64])

	cookies := resp.Cookies()
	for _, v := range cookies {
		ss := strings.Split(v.String(), ";")
		for _, c := range ss {
			if strings.Index(c, "csrftoken") >= 0 {
				csrftoken = strings.Split(c, "=")[1]
				return
			}
		}
	}
	fmt.Println("cannot get csrftoken")
}

func getLatestID() {
	fullURL := fmt.Sprintf("http://101weiqi.com/chessbook/")
	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		fmt.Println("Could not parse get chessbook page request:", err)
		return
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 6.1; WOW64; rv:45.0) Gecko/20100101 Firefox/45.0")
	req.Header.Set("Accept", "*/*")

	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Could not send get chessbook page request:", err)
		return
	}

	defer resp.Body.Close()
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("could not read chessbook page")
		return
	}

	keyword := "/chessbook/chess/"
	startPos := bytes.Index(data, []byte(keyword))
	if startPos < 0 {
		fmt.Println("can't find keyword", keyword, string(data))
		return
	}
	s := bytes.Index(data[startPos+len(keyword):], []byte("/"))
	id := data[startPos+len(keyword) : startPos+len(keyword)+s]
	latestID, _ = strconv.Atoi(string(id))
}

func main() {
	client = &http.Client{
		Timeout: 30 * time.Second,
	}

	flag.StringVar(&saveFileEncoding, "encoding", "gbk", "save SGF file encoding")
	flag.BoolVar(&quitIfExists, "q", false, "quit if the target file exists")
	flag.IntVar(&latestID, "l", 0, "the latest pid")
	flag.IntVar(&earliestID, "e", 1, "the earliest pid")
	flag.IntVar(&parallelCount, "p", 20, "the parallel routines count")
	flag.Parse()
	if latestID == 0 {
		getLatestID()
	}
	if latestID-earliestID < parallelCount {
		parallelCount = latestID - earliestID
	}
	getCSRF()
	fmt.Println("save SGF file encoding", saveFileEncoding)
	fmt.Println("quit if the target file exists", quitIfExists)
	fmt.Println("the latest pid", latestID)
	fmt.Println("the earliest pid", earliestID)
	fmt.Println("the parallel routines count", parallelCount)
	fmt.Println("csrf middleware token", csrfmiddlewaretoken)
	fmt.Println("csrf token", csrftoken)
	s := semaphore.NewSemaphore(parallelCount)
	for i := latestID; i >= earliestID && !quit; i-- {
		s.Acquire()
		go download(i, s)
	}

	wg.Wait()
	fmt.Println("Totally downloaded", downloadCount, " SGF files")
}
