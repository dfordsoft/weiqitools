package lol

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"kifudl/ic"
	"kifudl/semaphore"
	"kifudl/util"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var (
	client *http.Client
)

type Lol struct {
	sync.WaitGroup
	Sem                 *semaphore.Semaphore
	csrfmiddlewaretoken string
	csrftoken           string
	quit                bool // assume it's false as initial value
	QuitIfExists        bool
	SaveFileEncoding    string
	LatestID            int
	EarliestID          int
	DownloadCount       int32
}

func (l *Lol) getContent(path string) []byte {
	fullURL := fmt.Sprintf("http://101weiqi.com%s", path)
	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		log.Println("Could not parse get kifu request:", err)
		return []byte("")
	}

	req.Header.Set("Referer", "http://101weiqi.com")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 6.1; WOW64; rv:45.0) Gecko/20100101 Firefox/45.0")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("accept-language", `en-US,en;q=0.8`)
	req.Header.Set("cookie", fmt.Sprintf("csrftoken=%s", l.csrftoken))

	resp, err := client.Do(req)
	if err != nil {
		log.Println("Could not send get kifu request:", err)
		return []byte("")
	}

	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		log.Println("kifu request not 200")
		return []byte("")
	}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("can't read kifu content", err)
		return []byte("")
	}
	if l.SaveFileEncoding != "utf-8" {
		data = bytes.Replace(data, []byte("utf-8"), []byte(l.SaveFileEncoding), 1)
	}
	return data[3:] // remove BOM
}

type KifuPathResponse struct {
	Result int    `json:"result"`
	PURL   string `json:"purl"`
}

func (l *Lol) getPath(index int) string {
	// login
	data := fmt.Sprintf(`pid=%d&csrfmiddlewaretoken=%s`, index, l.csrfmiddlewaretoken)
	req, err := http.NewRequest("POST", "http://www.101weiqi.com/chessbook/download_sgf/", bytes.NewBufferString(data))
	if err != nil {
		log.Println("Could not parse get kifu path request:", err)
		return ""
	}

	req.Header.Set("Referer", fmt.Sprintf("http://www.101weiqi.com/chessbook/chess/%d/", index))
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 6.1; WOW64; rv:45.0) Gecko/20100101 Firefox/45.0")
	req.Header.Set("Accept", "application/json, text/javascript, */*; q=0.01")
	req.Header.Set("X-Requested-With", `XMLHttpRequest`)
	req.Header.Set("Content-Type", `application/x-www-form-urlencoded; charset=UTF-8`)
	req.Header.Set("cookie", fmt.Sprintf("csrftoken=%s", l.csrftoken))

	resp, err := client.Do(req)
	if err != nil {
		log.Println("Could not send get kifu path request:", err)
		return ""
	}

	defer resp.Body.Close()
	d, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("can't read kifu path")
		return ""
	}
	var m KifuPathResponse
	err = json.Unmarshal(d, &m)
	if err != nil {
		log.Println("can't unmarshal json", string(d), err)
		return ""
	}
	return m.PURL
}

func (l *Lol) download(index int) {
	l.Add(1)
	defer func() {
		l.Sem.Release()
		l.Done()
	}()
	tryGettingPath := 1
startGettingPath:
	p := l.getPath(index)
	if len(p) > 0 {
		if l.quit {
			return
		}
		if strings.Index(p, `小围`) > 0 || strings.Index(p, `大围`) > 0 || strings.Index(p, `老围`) > 0 {
			return
		}
		fullPath := "101weiqi/" + p[1:]
		if !util.Exists(fullPath) {
			tryGettingContent := 1
		startGettingContent:
			if l.quit {
				return
			}
			kifu := l.getContent(p)
			if len(kifu) > 0 {
				if bytes.Index(kifu, []byte("EV[]")) > 0 {
					log.Println("empty ev node for", fullPath)
					return
				}
				dir, _ := filepath.Split(fullPath)
				if !util.Exists(dir) {
					os.MkdirAll(dir, 0777)
				}
				if l.SaveFileEncoding != "utf-8" {
					kifu = ic.Convert("utf-8", l.SaveFileEncoding, kifu)
				}
				// save to file
				ioutil.WriteFile(fullPath, kifu, 0644)
				kifu = nil
				atomic.AddInt32(&l.DownloadCount, 1)
			} else {
				tryGettingContent++
				if tryGettingContent < 10 {
					time.Sleep(3 * time.Second)
					goto startGettingContent
				}
			}
		} else {
			if l.QuitIfExists {
				log.Println(fullPath, "exists, quit now")
				l.quit = true
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

func (l *Lol) getCSRF() {
	fullURL := fmt.Sprintf("http://101weiqi.com/chessbook/chess/%d", l.LatestID)
	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		log.Println("Could not parse get latest ID page request:", err)
		return
	}

	req.Header.Set("Referer", "http://101weiqi.com")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 6.1; WOW64; rv:45.0) Gecko/20100101 Firefox/45.0")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("accept-language", `en-US,en;q=0.8`)

	resp, err := client.Do(req)
	if err != nil {
		log.Println("Could not send get latest ID page request:", err)
		return
	}

	defer resp.Body.Close()
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("could not read latest ID page")
		return
	}

	startPos := bytes.Index(data, []byte("csrfmiddlewaretoken"))
	if startPos < 0 {
		log.Println("could not find csrfmiddlewaretoken")
		return
	}
	csrfmwtokenPos := bytes.Index(data[startPos:], []byte("value='"))

	l.csrfmiddlewaretoken = string(data[startPos+csrfmwtokenPos+7 : startPos+csrfmwtokenPos+7+64])

	cookies := resp.Cookies()
	for _, v := range cookies {
		ss := strings.Split(v.String(), ";")
		for _, c := range ss {
			if strings.Index(c, "csrftoken") >= 0 {
				l.csrftoken = strings.Split(c, "=")[1]
				return
			}
		}
	}
	log.Println("cannot get csrftoken")
}

func (l *Lol) getLatestID() {
	fullURL := fmt.Sprintf("http://101weiqi.com/chessbook/")
	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		log.Println("Could not parse get chessbook page request:", err)
		return
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 6.1; WOW64; rv:45.0) Gecko/20100101 Firefox/45.0")
	req.Header.Set("Accept", "*/*")

	resp, err := client.Do(req)
	if err != nil {
		log.Println("Could not send get chessbook page request:", err)
		return
	}

	defer resp.Body.Close()
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("could not read chessbook page")
		return
	}

	keyword := "/chessbook/chess/"
	startPos := bytes.Index(data, []byte(keyword))
	if startPos < 0 {
		log.Println("can't find keyword", keyword, string(data))
		return
	}
	s := bytes.Index(data[startPos+len(keyword):], []byte("/"))
	id := data[startPos+len(keyword) : startPos+len(keyword)+s]
	l.LatestID, _ = strconv.Atoi(string(id))
}

func (l *Lol) Download(w *sync.WaitGroup) {
	w.Add(1)
	defer w.Done()
	client = &http.Client{
		Timeout: 30 * time.Second,
	}

	if l.LatestID == 0 {
		l.getLatestID()
	}

	l.getCSRF()

	fmt.Println("the latest pid", l.LatestID)
	fmt.Println("the earliest pid", l.EarliestID)
	fmt.Println("csrf middleware token", l.csrfmiddlewaretoken)
	fmt.Println("csrf token", l.csrftoken)

	for i := l.LatestID; i >= l.EarliestID && !l.quit; i-- {
		l.Sem.Acquire()
		go l.download(i)
	}

	l.Wait()
	fmt.Println("Totally downloaded", l.DownloadCount, " SGF files")
}
