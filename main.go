package main

import (
	"os"
	"sync"

	"./adapter"
	"gopkg.in/mgo.v2"
)

func main() {
	proj_ids := []int{11, 12, 14, 15, 17, 19, 23, 27, 303, 34, 191, 197, 202, 205, 208, 228, 229, 231, 235, 244, 247, 248, 259, 261, 340, 36, 39, 413, 43, 511, 598, 603, 695, 717}
	//proj_ids := []int{511}

	session, err := mgo.Dial("172.17.0.2")
	if err != nil {
		panic(err)
	}
	defer session.Close()

	mrc := session.DB("mrm").C("mergerequests")

	gitLabAdapter := adapter.NewGitLabAdapter(os.Getenv("GITLAB_API_KEY"), session)

	var wg sync.WaitGroup
	for _, projId := range proj_ids {
		wg.Add(1)
		go gitLabAdapter.LoadMergeRequests(projId, mrc, &wg)
	}
	wg.Wait()
}
