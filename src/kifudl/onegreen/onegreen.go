package onegreen

import (
	"bytes"
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

type Page struct {
	URL   string
	Count int
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

	req.Header.Set("Referer", "http://game.onegreen.net/weiqi/ShowClass.asp?ClassID=1218&page=1254")
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

	// extract SGF data
	index := bytes.Index(kifu, []byte(`sgftext=`))
	if index < 0 {
		fmt.Println("cannot find start keyword")
		return
	}
	kifu = kifu[index+8:]
	index = bytes.Index(kifu, []byte(`" ALLOWSCRIPTACCESS=`))
	if index < 0 {
		fmt.Println("cannot find end keyword")
		return
	}
	kifu = kifu[:index]

	u, err := url.Parse(sgf)
	if err != nil {
		log.Fatal(err)
	}
	fullPath := u.Path[1:]
	fullPath = strings.Replace(fullPath, ".html", ".sgf", -1)
	insertPos := len(fullPath) - 7
	fullPathByte := []byte(fullPath)
	fullPathByte = append(fullPathByte[:insertPos], append([]byte{'/'}, fullPathByte[insertPos:]...)...)
	fullPath = string(fullPathByte)
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

func downloadPage(page string, s *semaphore.Semaphore) {
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
		return
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 6.1; WOW64; rv:45.0) Gecko/20100101 Firefox/45.0")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
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

	regex := regexp.MustCompile(`href='(http:\/\/game\.onegreen\.net\/weiqi\/HTML\/[0-9a-zA-Z\-\_]+\.html)'`)
	ss := regex.FindAllSubmatch(data, -1)
	for _, match := range ss {
		if quit {
			break
		}
		sgf := string(match[1])
		go downloadKifu(sgf, s)
	}
}

func download(w *sync.WaitGroup) {
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

	pagelist := []Page{
		{"http://game.onegreen.net/weiqi/ShowClass.asp?ClassID=1218&page=%d", 1254},
		{"http://game.onegreen.net/weiqi/ShowClass.asp?ClassID=1223&page=%d", 514},
	}

	s := semaphore.NewSemaphore(parallelCount)
	for _, page := range pagelist {
		if quit {
			break
		}
		for i := 1; !quit && i <= page.Count; i++ {
			u := fmt.Sprintf(page.URL, i)
			downloadPage(u, s)
		}
	}

	wg.Wait()
	fmt.Println("Totally downloaded", downloadCount, " SGF files")
}
