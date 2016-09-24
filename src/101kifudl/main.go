package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var (
	wg                  sync.WaitGroup
	client              *http.Client
	csrfmiddlewaretoken string
	csrftoken           string
	quitIfExists        = true
	quit                = false
	runIconvAfterSave   = false
)

type Semaphore struct {
	c chan int
}

func NewSemaphore(n int) *Semaphore {
	s := &Semaphore{
		c: make(chan int, n),
	}
	return s
}

func (s *Semaphore) Acquire() {
	s.c <- 0
}

func (s *Semaphore) Release() {
	<-s.c
}

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
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("can't read kifu content", err)
		return []byte("")
	}
	// convert to gbk, not work
	// var b bytes.Buffer
	// wInUTF8 := transform.NewWriter(&b, simplifiedchinese.GBK.NewEncoder())
	// _, err = wInUTF8.Write([]byte(string([]rune(string(data)))))
	// wInUTF8.Close()
	// if err != nil {
	// 	fmt.Println("can't convert kifu encoding from utf-8 to gbk", err)
	// 	return []byte("")
	// }
	data = bytes.Replace(data, []byte("utf-8"), []byte("gbk"), 1)
	if runIconvAfterSave {
		return data[3:]
	}
	return data
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

func exists(f string) bool {
	stat, err := os.Stat(f)
	if err == nil {
		if stat.Mode()&os.ModeType == 0 {
			return true
		}
		return false
	}
	if os.IsNotExist(err) {
		return false
	}
	return false
}

func download(index int, s *Semaphore) {
	defer s.Release()
	wg.Add(1)
	defer wg.Done()
	tryGettingPath := 1
startGettingPath:
	p := getPath(index)
	if len(p) > 0 {
		if strings.Index(p, `小围`) > 0 || strings.Index(p, `大围`) > 0 || strings.Index(p, `老围`) > 0 {
			return
		}
		fullPath := p[1:]
		if !exists(fullPath) {
			tryGettingContent := 1
		startGettingContent:
			kifu := getContent(p)
			if len(kifu) > 0 {
				if bytes.Index(kifu, []byte("EV[]")) > 0 {
					return
				}
				dir, _ := filepath.Split(fullPath)
				if !exists(dir) {
					os.MkdirAll(dir, 0777)
				}
				// save to file
				ioutil.WriteFile(fullPath, kifu, 0644)
				kifu = nil
				// convert from utf-8 to gbk
				if runIconvAfterSave {
					cmd := exec.Command("iconv", "-f", "utf-8", "-t", "gbk", fullPath)
					stdout, err := cmd.StdoutPipe()
					if err != nil {
						fmt.Println("connecting stdout pipe failed", err)
						return
					}
					if err = cmd.Start(); err != nil {
						fmt.Println("starting iconv command failed", err)
						return
					}
					d, err := ioutil.ReadAll(stdout)
					if err != nil {
						fmt.Println("reading stdout failed", err)
					}
					if err = cmd.Wait(); err != nil {
						fmt.Println("waiting for iconv command exiting failed", err)
					}
					ioutil.WriteFile(fullPath, d, 0644)
				}
			} else {
				tryGettingContent++
				if tryGettingContent < 10 {
					time.Sleep(3 * time.Second)
					goto startGettingContent
				}
			}
		} else {
			if !quitIfExists {
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

func main() {
	client = &http.Client{
		Timeout: 30 * time.Second,
	}
	latestID := 218313
	earliestID := 1
	parallelCount := 20
	flag.BoolVar(&runIconvAfterSave, "iconv", false, "run iconv after file saved")
	flag.BoolVar(&quitIfExists, "q", true, "quit if the target file exists")
	flag.IntVar(&latestID, "l", 218313, "the latest pid")
	flag.IntVar(&earliestID, "e", 1, "the earliest pid")
	flag.IntVar(&parallelCount, "p", 20, "the parallel routines count")
	flag.StringVar(&csrfmiddlewaretoken, "m", "IEFLXplkObujFxDBpg8tpaF18s8igd07IKgUTHq4cynnXVAF4u3agM8BXUOS9861", "csrf middleware token")
	flag.StringVar(&csrftoken, "t", "kbkR6DR5lDOxCRQyXhTPMT6aWIw5wM2fkhV02VWPJ0HBUfNCCvOwDvzKLacFpH89", "csrf token as cookie")
	flag.Parse()
	fmt.Println("run iconv after file saved", runIconvAfterSave)
	fmt.Println("quit if the target file exists", quitIfExists)
	fmt.Println("the latest pid", latestID)
	fmt.Println("the earliest pid", earliestID)
	fmt.Println("the parallel routines count", parallelCount)
	fmt.Println("csrf middleware token", csrfmiddlewaretoken)
	fmt.Println("csrf token", csrftoken)
	s := NewSemaphore(parallelCount)
	for i := latestID; i >= earliestID && !quit; i-- {
		s.Acquire()
		go download(i, s)
	}

	wg.Wait()
}
