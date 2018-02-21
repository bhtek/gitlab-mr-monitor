package main

import (
	"sync"

	"./adapter"
	"gopkg.in/mgo.v2"
)

func main() {
	//projs, err := adapter.SearchProjects()
	//if err != nil {
	//	log.Fatal(err)
	//}
	//
	//filters := []string{"aladdin", "tms"}
	//
	//for _, proj := range projs {
	//	selected := false
	//	if strings.Index(proj.NameWithNamespace, "consul") < 0 {
	//	FilterLoop:
	//		for _, filter := range filters {
	//			if strings.Index(proj.NameWithNamespace, filter) > -1 {
	//				selected = true
	//				break FilterLoop
	//			}
	//		}
	//	}
	//
	//	if selected {
	//		fmt.Println(proj)
	//	}
	//}

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
