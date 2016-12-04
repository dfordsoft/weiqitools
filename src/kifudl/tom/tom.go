package tom

import (
	"flag"
	"fmt"
	"ic"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
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
	saveFileEncoding string
	quit             bool // assume it's false as initial value
	quitIfExists     bool
	parallelCount    int
	downloadCount    int32
)

func getNextPageURL(page string) string {
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
			fmt.Println("converting", n, "to number failed", err)
			return p
		}
		number++
		p = fmt.Sprintf("%s%2.2d.html", p[:index+1], number)
	}
	return p
}

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

	u, _ := url.Parse(sgf)
	req.Header.Set("Referer", fmt.Sprintf("%s://%s", u.Scheme, u.Host))
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 6.1; WOW64; rv:45.0) Gecko/20100101 Firefox/45.0")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
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

func downloadPage(page string, s *semaphore.Semaphore) bool {
	wg.Add(1)
	s.Acquire()
	defer func() {
		s.Release()
		wg.Done()
	}()
	retry := 0
	req, err := http.NewRequest("GET", page, nil)
	if err != nil {
		fmt.Println("Could not parse page request:", err)
		return false
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 6.1; WOW64; rv:45.0) Gecko/20100101 Firefox/45.0")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
doPageRequest:
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Could not send page request:", err)
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
		fmt.Println("cannot read page content", err)
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
		if quit {
			break
		}
		sgf := string(match[1])
		sgf = strings.Replace(sgf, "../..", fmt.Sprintf("%s://%s", u.Scheme, u.Host), 1)
		sgf = strings.Replace(sgf, "./..", fmt.Sprintf("%s://%s", u.Scheme, u.Host), 1)
		sgf = strings.Replace(sgf, "..", fmt.Sprintf("%s://%s", u.Scheme, u.Host), 1)
		sgf = strings.Replace(sgf, "weiqi.cn.tom.com", "weiqi.tom.com", 1)
		sgf = strings.Replace(sgf, "weiqi.sports.tom.comcom", "weiqi.sports.tom.com", 1)
		go downloadKifu(sgf, s)
	}

	return true
}

func Download(w *sync.WaitGroup) {
	w.Add(1)
	defer w.Done()
	client = &http.Client{
		Timeout: 60 * time.Second,
	}
	flag.StringVar(&saveFileEncoding, "encoding", "gbk", "save SGF file encoding")
	flag.BoolVar(&quitIfExists, "q", true, "quit if the target file exists")
	flag.IntVar(&parallelCount, "p", 20, "the parallel routines count")
	flag.Parse()

	fmt.Println("save SGF file encoding", saveFileEncoding)
	fmt.Println("quit if the target file exists", quitIfExists)

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

	s := semaphore.NewSemaphore(parallelCount)
	for _, page := range pagelist {
		p := page
		for !quit && downloadPage(p, s) {
			p = getNextPageURL(p)
		}
	}

	wg.Wait()
	fmt.Println("Totally downloaded", downloadCount, " SGF files")
}
