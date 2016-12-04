package main

import (
    "sync"
    "kifudl/lol"
)

func main() {
	wg               sync.WaitGroup
    lol.download(&wg)

    wg.Wait()
}
