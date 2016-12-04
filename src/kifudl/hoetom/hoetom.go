package hoetom

import (
	"fmt"
	"ic"
	"io/ioutil"
	"kifudl/semaphore"
	"kifudl/util"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var (
	wg     sync.WaitGroup
	client *http.Client
)

type Hoetom struct {
	sessionID        string
	userID           string
	password         string
	passwordMd5      string
	SaveFileEncoding string
	quit             bool // assume it's false as initial value
	QuitIfExists     bool
	LatestPageID     int
	EarliestPageID   int
	ParallelCount    int
	DownloadCount    int32
}

func (h *Hoetom) getSessionID() {
	fullURL := fmt.Sprintf("http://www.hoetom.com/servlet/login")
	postBody := fmt.Sprintf("userid=%s&passwd=%s&passwdmd5=%s", h.userID, h.password, h.passwordMd5)
	req, err := http.NewRequest("POST", fullURL, strings.NewReader(postBody))
	if err != nil {
		log.Println("Could not parse login request:", err)
		return
	}

	req.Header.Set("Origin", "http://www.hoetom.com")
	req.Header.Set("Referer", "http://www.hoetom.com/index.jsp")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 6.1; WOW64; rv:45.0) Gecko/20100101 Firefox/45.0")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("accept-language", `en-US,en;q=0.8`)
	req.Header.Set("Cache-Control", "max-age=0")

	resp, err := client.Do(req)
	if err != nil {
		log.Println("Could not send login request:", err)
		return
	}

	defer resp.Body.Close()

	cookies := resp.Cookies()
	for _, v := range cookies {
		ss := strings.Split(v.String(), ";")
		for _, c := range ss {
			if strings.Index(c, "JSESSIONID") >= 0 {
				h.sessionID = strings.Split(c, "=")[1]
				return
			}
		}
	}
	log.Println("cannot get session id")
}

func (h *Hoetom) downloadKifu(id int, s *semaphore.Semaphore) {
	wg.Add(1)
	s.Acquire()
	defer func() {
		s.Release()
		wg.Done()
	}()
	if h.quit {
		return
	}
	retry := 0
	fullURL := fmt.Sprintf("http://www.hoetom.com/chessmanual.jsp?id=%d", id)
	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		log.Println("Could not parse kifu request:", err)
		return
	}

	req.Header.Set("Origin", "http://www.hoetom.com")
	req.Header.Set("Referer", "http://www.hoetom.com/matchlatest_pro.jsp")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 6.1; WOW64; rv:45.0) Gecko/20100101 Firefox/45.0")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("accept-language", `en-US,en;q=0.8`)
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	req.Header.Set("cookie", fmt.Sprintf("JSESSIONID=%s; userid=%s", h.sessionID, h.userID))
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
		log.Println("kifu request not 200:", resp.StatusCode, fullURL)
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
	ss := strings.Split(resp.Header.Get("Content-Disposition"), ";")
	if len(ss) < 2 {
		log.Println("cannot get content-disposition")
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
	fullPath := fmt.Sprintf("hoetom/%s/%s", dir, filename)
	if util.Exists(fullPath) {
		if h.QuitIfExists {
			h.quit = true
		}
		return
	}
	if h.SaveFileEncoding != "gbk" {
		kifu = ic.Convert("gbk", h.SaveFileEncoding, kifu)
	}
	ioutil.WriteFile(fullPath, kifu, 0644)
	kifu = nil
	atomic.AddInt32(&h.DownloadCount, 1)
}

func (h *Hoetom) downloadPage(page int, s *semaphore.Semaphore) {
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
		log.Println("Could not parse page request:", err)
		return
	}

	req.Header.Set("Origin", "http://www.hoetom.com")
	req.Header.Set("Referer", "http://www.hoetom.com/matchlatest_pro.jsp")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 6.1; WOW64; rv:45.0) Gecko/20100101 Firefox/45.0")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("accept-language", `en-US,en;q=0.8`)
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	req.Header.Set("cookie", fmt.Sprintf("JSESSIONID=%s; userid=%s", h.sessionID, h.userID))
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

	regex := regexp.MustCompile(`matchinfor\.jsp\?id=([0-9]+)`)
	ss := regex.FindAllSubmatch(data, -1)
	for _, match := range ss {
		if h.quit {
			break
		}
		id, err := strconv.Atoi(string(match[1]))
		if err != nil {
			log.Printf("converting %s to number failed", string(match[1]))
			continue
		}

		go h.downloadKifu(id, s)
	}
}

func (h *Hoetom) Download(w *sync.WaitGroup) {
	w.Add(1)
	defer w.Done()
	client = &http.Client{
		Timeout: 30 * time.Second,
	}

	h.getSessionID()
	fmt.Println("save SGF file encoding", h.SaveFileEncoding)
	fmt.Println("quit if the target file exists", h.QuitIfExists)
	fmt.Println("the latest pid", h.LatestPageID)
	fmt.Println("the earliest pid", h.EarliestPageID)
	fmt.Println("the parallel routines count", h.ParallelCount)
	fmt.Println("session id", h.sessionID)
	s := semaphore.NewSemaphore(h.ParallelCount)
	for i := h.LatestPageID; i <= h.EarliestPageID && !h.quit; i++ {
		h.downloadPage(i, s)
	}

	wg.Wait()
	fmt.Println("Totally downloaded", h.DownloadCount, " SGF files")
}
