package adapter

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	url2 "net/url"
	"strconv"
	"sync"
	"time"
)

const (
	baseUrl = "http://git.sg.m-daq.net/api/v4"
)

type GitLabAdapter struct {
	ApiKey string
}

type GitLabProj struct {
	Id   uint32
	Name string
}

func (gla GitLabAdapter) SearchProjects() ([]GitLabProj, error) {
	url, err := url2.Parse(baseUrl + "/projects")
	if err != nil {
		return nil, err
	}

	client := &http.Client{
		Timeout: time.Second * 60,
	}

	var projects []GitLabProj
	var wg sync.WaitGroup
	projsChan := make(chan []GitLabProj)
	wg.Add(1)
	go gla.fetch(client, url, -1, projsChan, &wg)

	go func() {
		wg.Wait()
		close(projsChan)
		fmt.Println("Closed channel.")
	}()

	for projs := range projsChan {
		projects = append(projects, projs...)
	}

	return projects, nil
}

func (gla GitLabAdapter) fetch(client *http.Client, url *url2.URL, page int, projsChan chan []GitLabProj, wg *sync.WaitGroup) {
	q := url.Query()

	if page > 0 {
		q.Set("page", strconv.Itoa(page))
	}
	//q.Add("search", "lambda")

	url.RawQuery = q.Encode()

	fmt.Println("boon ", url.String())

	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		log.Fatal(err)
	}

	req.Header.Add("PRIVATE-TOKEN", gla.ApiKey)

	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}

	if page < 0 {
		totalPagesHeader := resp.Header["X-Total-Pages"]
		//fmt.Printf("%s: %d", totalPagesHeader, len(totalPagesHeader))
		if totalPagesHeader != nil && len(totalPagesHeader) > 0 && len(totalPagesHeader[0]) > 0 {
			totalPages, err := strconv.Atoi(totalPagesHeader[0])
			if err != nil {
				log.Fatal(err)
			}

			for p := 2; p <= totalPages; p++ {
				wg.Add(1)
				go gla.fetch(client, url, p, projsChan, wg)
			}
		}
	}
	//for k, v := range resp.Header {
	//	fmt.Printf("%s: %s\n", k, v)
	//}

	response, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	var projects []GitLabProj
	err = json.Unmarshal(response, &projects)
	if err != nil {
		log.Fatal(err)
	}

	projsChan <- projects

	fmt.Println("Done for call of page ", page)
	wg.Done()
}
