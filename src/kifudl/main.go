package main

import (
	"kifudl/hoetom"
	"kifudl/lol"
	"kifudl/onegreen"
	"kifudl/sina"
	"kifudl/tom"
	"kifudl/weiqitv"
	"kifudl/xgoo"
	"sync"
)

var (
	wg sync.WaitGroup
)

func main() {
	lol.Download(&wg)
	hoetom.Download(&wg)
	sina.Download(&wg)
	tom.Download(&wg)
	xgoo.Download(&wg)
	weiqitv.Download(&wg)
	onegreen.Download(&wg)

	wg.Wait()
}
