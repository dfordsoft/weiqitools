package sina

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

type Sina struct {
	sync.WaitGroup
	Sem              *semaphore.Semaphore
	SaveFileEncoding string
	quit             bool // assume it's false as initial value
	QuitIfExists     bool
	LatestPageID     int
	EarliestPageID   int
	DownloadCount    int32
}

func (s *Sina) downloadKifu(sgf string) {
	s.Add(1)
	s.Sem.Acquire()
	defer func() {
		s.Sem.Release()
		s.Done()
	}()
	if s.quit {
		return
	}
	retry := 0

	req, err := http.NewRequest("GET", sgf, nil)
	if err != nil {
		log.Println("Could not parse kifu request:", err)
		return
	}

	req.Header.Set("Referer", "http://duiyi.sina.com.cn/gibo/new_gibo.asp?cur_page=689")
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
	fullPath := "sina/" + u.Path[1:]
	if util.Exists(fullPath) {
		if s.QuitIfExists {
			s.quit = true
		}
		return
	}

	dir := path.Dir(fullPath)
	if !util.Exists(dir) {
		os.MkdirAll(dir, 0777)
	}
	if s.SaveFileEncoding != "gbk" {
		kifu = ic.Convert("gbk", s.SaveFileEncoding, kifu)
	}
	ioutil.WriteFile(fullPath, kifu, 0644)
	kifu = nil
	atomic.AddInt32(&s.DownloadCount, 1)
}

func (s *Sina) downloadPage(page int) {
	s.Add(1)
	s.Sem.Acquire()
	defer func() {
		s.Sem.Release()
		s.Done()
	}()
	retry := 0
	fullURL := fmt.Sprintf("http://duiyi.sina.com.cn/gibo/new_gibo.asp?cur_page=%d", page)
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

	regex := regexp.MustCompile(`JavaScript:gibo_load\('(http:\/\/duiyi\.sina\.com\.cn\/cgibo\/[0-9]+\/[0-9a-zA-Z\-]+\.sgf)'\)`)
	ss := regex.FindAllSubmatch(data, -1)
	dl := make(map[string]bool, len(ss))
	for _, match := range ss {
		if s.quit {
			break
		}
		sgf := string(match[1])
		if _, ok := dl[sgf]; ok {
			continue
		}
		dl[sgf] = true
		go s.downloadKifu(sgf)
	}

	regex = regexp.MustCompile(`JavaScript:gibo_load\('(http:\/\/duiyi\.sina\.com\.cn\/cgibo\/[0-9a-zA-Z\-]+\.sgf)'\)`)
	ss = regex.FindAllSubmatch(data, -1)
	dl = make(map[string]bool, len(ss))
	for _, match := range ss {
		if s.quit {
			break
		}
		sgf := string(match[1])
		if _, ok := dl[sgf]; ok {
			continue
		}
		dl[sgf] = true
		go s.downloadKifu(sgf)
	}
	dl = nil
}

func (s *Sina) Download(w *sync.WaitGroup) {
	w.Add(1)
	defer w.Done()
	client = &http.Client{
		Timeout: 30 * time.Second,
	}

	fmt.Println("sina the latest pid", s.LatestPageID)
	fmt.Println("sina the earliest pid", s.EarliestPageID)

	for i := s.LatestPageID; i <= s.EarliestPageID && !s.quit; i++ {
		s.downloadPage(i)
	}

	s.Wait()
	fmt.Println("Totally downloaded", s.DownloadCount, " SGF files from Sina")
}
