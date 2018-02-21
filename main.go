package main

import (
	"sync"

	"./adapter"
	"gopkg.in/mgo.v2"
)

func main() {
	proj_ids := []int{11, 12, 14, 15, 17, 19, 23, 27, 303, 34, 202, 228, 231, 235, 340, 36, 39, 413, 43, 511, 603}
	//proj_ids := []int{511}

	session, err := mgo.Dial("172.17.0.2")
	if err != nil {
		panic(err)
	}
	defer session.Close()

	mrc := session.DB("mrm").C("mergerequests")

	adptr := adapter.NewGitLabAdapter("By3ySyMr-9S5yD3zDYsu", session)

	var wg sync.WaitGroup
	for _, proj_id := range proj_ids {
		wg.Add(1)
		go adptr.LoadMergeRequests(proj_id, mrc, &wg)
	}
	wg.Wait()
}
