package tom

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
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var (
	client *http.Client
)

type Tom struct {
	sync.WaitGroup
	Sem              *semaphore.Semaphore
	SaveFileEncoding string
	quit             bool // assume it's false as initial value
	QuitIfExists     bool
	DownloadCount    int32
}

func (t *Tom) getNextPageURL(page string) string {
	p := page
	index := strings.Index(page, "_")
	if index < 0 {
		// insert _02 before last .html
		i := strings.LastIndex(p, ".html")
		o := []byte(p)
		o = append(o[:i], append([]byte("_02"), o[i:]...)...)
		p = string(o)
	} else {
		// extract number and increase 1
		i := strings.LastIndex(p, ".html")
		n := p[index+1 : i]
		number, err := strconv.Atoi(n)
		if err != nil {
			log.Println("converting", n, "to number failed", err)
			return p
		}
		number++
		p = fmt.Sprintf("%s%2.2d.html", p[:index+1], number)
	}
	return p
}

func (t *Tom) downloadKifu(sgf string) {
	t.Add(1)
	t.Sem.Acquire()
	defer func() {
		t.Sem.Release()
		t.Done()
	}()
	if t.quit {
		return
	}
	retry := 0

	req, err := http.NewRequest("GET", sgf, nil)
	if err != nil {
		log.Println("Could not parse kifu request:", err)
		return
	}

	u, _ := url.Parse(sgf)
	req.Header.Set("Referer", fmt.Sprintf("%s://%s", u.Scheme, u.Host))
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

	fullPath := "tom/" + u.Path[1:]
	if util.Exists(fullPath) {
		if t.QuitIfExists {
			t.quit = true
		}
		return
	}

	dir := path.Dir(fullPath)
	if !util.Exists(dir) {
		os.MkdirAll(dir, 0777)
	}
	if t.SaveFileEncoding != "gbk" {
		kifu = ic.Convert("gbk", t.SaveFileEncoding, kifu)
	}
	ioutil.WriteFile(fullPath, kifu, 0644)
	kifu = nil
	atomic.AddInt32(&t.DownloadCount, 1)
}

func (t *Tom) downloadPage(page string) bool {
	t.Add(1)
	t.Sem.Acquire()
	defer func() {
		t.Sem.Release()
		t.Done()
	}()
	retry := 0
	req, err := http.NewRequest("GET", page, nil)
	if err != nil {
		log.Println("Could not parse page request:", err)
		return false
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 6.1; WOW64; rv:45.0) Gecko/20100101 Firefox/45.0")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
doPageRequest:
	resp, err := client.Do(req)
	if err != nil {
		log.Println("Could not send page request:", err)
		retry++
		if retry < 3 {
			time.Sleep(3 * time.Second)
			goto doPageRequest
		}
		return false
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
		return false
	}

	regex := regexp.MustCompile(`href="([a-zA-Z0-9:\/\-\.]+\.sgf)"`)
	ss := regex.FindAllSubmatch(data, -1)

	if len(ss) == 0 {
		return false
	}

	u, _ := url.Parse(page)
	for _, match := range ss {
		if t.quit {
			break
		}
		sgf := string(match[1])
		sgf = strings.Replace(sgf, "../..", fmt.Sprintf("%s://%s", u.Scheme, u.Host), 1)
		sgf = strings.Replace(sgf, "./..", fmt.Sprintf("%s://%s", u.Scheme, u.Host), 1)
		sgf = strings.Replace(sgf, "..", fmt.Sprintf("%s://%s", u.Scheme, u.Host), 1)
		sgf = strings.Replace(sgf, "weiqi.cn.tom.com", "weiqi.tom.com", 1)
		sgf = strings.Replace(sgf, "weiqi.sports.tom.comcom", "weiqi.sports.tom.com", 1)
		go t.downloadKifu(sgf)
	}

	return true
}

func (t *Tom) Download(w *sync.WaitGroup) {
	w.Add(1)
	defer w.Done()
	client = &http.Client{
		Timeout: 60 * time.Second,
	}

	pagelist := []string{
		"http://weiqi.tom.com/php/listqipu.html",
		"http://weiqi.sports.tom.com/php/listqipu2012.html",
		"http://weiqi.sports.tom.com/php/listqipu2011.html",
		"http://weiqi.sports.tom.com/php/listqipu2010.html",
		"http://weiqi.sports.tom.com/php/listqipu2009.html",
		"http://weiqi.sports.tom.com/php/listqipu2008.html",
		"http://weiqi.sports.tom.com/php/listqipu2007.html",
		"http://weiqi.sports.tom.com/php/listqipu2006.html",
		"http://weiqi.sports.tom.com/php/listqipu2005.html",
		"http://weiqi.sports.tom.com/php/listqipu2000.html",
	}

	for _, page := range pagelist {
		p := page
		for !t.quit && t.downloadPage(p) {
			p = t.getNextPageURL(p)
		}
	}

	t.Wait()
	fmt.Println("Totally downloaded", t.DownloadCount, " SGF files from Tom")
}
