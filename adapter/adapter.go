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

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

const (
	baseUrl = "http://git.sg.m-daq.net/api/v4"
)

type GitLabAdapter struct {
	ApiKey  string
	Session *mgo.Session
	client  *http.Client
}

func NewGitLabAdapter(apiKey string, session *mgo.Session) GitLabAdapter {
	client := &http.Client{
		Timeout: time.Second * 60,
	}
	return GitLabAdapter{apiKey, session, client}
}

type GitLabProj struct {
	Id                uint32
	Name              string
	NameWithNamespace string `json:"name_with_namespace"`
}

type GitLabMergeRequest struct {
	Id           uint32
	Iid          uint32
	ProjectId    uint32    `json:"project_id"`
	Title        string
	State        string
	Upvotes      int
	Downvotes    int
	CreatedAt    time.Time `json:"created_at"`
	TargetBranch string    `json:"target_branch"`
	SourceBranch string    `json:"source_branch"`
	WebUrl       string    `json:"web_url"`
	Author       GitLabUser
	Assignee     GitLabUser
	Emojis       []GitLabEmoji
}

type GitLabUser struct {
	Id       uint32
	Name     string
	Username string
}

type GitLabEmoji struct {
	Id            uint32
	Name          string
	User          GitLabUser
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	AwardableId   uint32    `json:"awardable_id"`
	AwardableType string    `json:"awardable_type"`
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

func (gla GitLabAdapter) LoadMergeRequests(projectId int, mrc *mgo.Collection, wg *sync.WaitGroup) {
	defer wg.Done()

	url, err := url2.Parse(baseUrl + "/projects/" + strconv.Itoa(projectId) + "/merge_requests?sort=asc")
	if err != nil {
		log.Fatal(err)
		return
	}

	lastMergeRequest := GitLabMergeRequest{}
	type list []interface{}
	var lastCreatedAt time.Time
	err = mrc.Find(bson.M{"projectid": projectId, "state": bson.M{"$nin": list{"closed", "merged"}}}).Sort("createdat").One(&lastMergeRequest)
	if err != nil {
		err = mrc.Find(bson.M{"projectid": projectId, "state": bson.M{"$in": list{"closed", "merged"}}}).Sort("-createdat").One(&lastMergeRequest)
	}
	if err != nil {
		fmt.Printf("Failed to find last merge for projectid=%d due to %s.\n\t%s\n", projectId, err, url)
	} else {
		lastCreatedAt = lastMergeRequest.CreatedAt
		fmt.Printf("Will look for merge requests for projectid=%d after %s.\n", projectId, lastCreatedAt)
	}

	query := url.Query()
	if !lastCreatedAt.IsZero() {
		query.Add("created_after", lastCreatedAt.String())
	}

	var pageNumber = 1
	var totalMods int
	for {
		mods, hasMore, err := gla.loadMergeRequestPage(pageNumber, url, &query, mrc)
		if err != nil {
			log.Fatal(err)
			return
		}

		totalMods += mods

		if !hasMore {
			break
		}

		pageNumber += 1
	}

	fmt.Printf("Handled %d merge request(s) for projectId=%d.\n", totalMods, projectId)
}

func (gla GitLabAdapter) loadMergeRequestPage(pageNumber int, url *url2.URL, query *url2.Values, mrc *mgo.Collection) (int, bool, error) {
	query.Set("page", strconv.Itoa(pageNumber))

	url.RawQuery = query.Encode()

	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return 0, false, err
	}

	req.Header.Add("PRIVATE-TOKEN", gla.ApiKey)

	resp, err := gla.client.Do(req)

	nextPage := resp.Header.Get("X-Next-Page")

	response, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	var mergeRequests []GitLabMergeRequest
	err = json.Unmarshal(response, &mergeRequests)
	if err != nil {
		log.Fatal(fmt.Sprintf("Failed to unmarshal for url=%s, err=%s\n", url, err))
		return 0, false, err
	}

	var mods = 0

	for _, mergeRequest := range mergeRequests {
		var mrdb GitLabMergeRequest
		err := mrc.Find(bson.M{"id": mergeRequest.Id}).One(&mrdb)
		if err != nil {
			if mergeRequest.State != "opened" {
				err = gla.enrichMergeRequest(&mergeRequest)
				if err != nil {
					return 0, false, err
				}
			}
			mods += 1
			mrc.Insert(mergeRequest)
			//fmt.Printf("%s\n", mergeRequest)
		} else {
			if mrdb.State == "opened" && mergeRequest.State != "opened" {
				gla.enrichMergeRequest(&mergeRequest)
				if err != nil {
					return 0, false, err
				}

				mods += 1
				mrc.Update(bson.M{"id": mergeRequest.Id}, mergeRequest)
				//fmt.Printf("Updated mr %d to state=%s\n", mergeRequest.Id, mergeRequest.State)
			} else {
				//fmt.Printf("Skipped mr %d @ state=%s\n", mergeRequest.Id, mergeRequest.State)
			}
		}
	}

	if nextPage != "" {
		return mods, true, nil
	} else {
		return mods, false, nil
	}
}

func (gla GitLabAdapter) enrichMergeRequest(mr *GitLabMergeRequest) error {
	url, err := url2.Parse(baseUrl + "/projects/" + fmt.Sprint(mr.ProjectId) + "/merge_requests/" +
		fmt.Sprint(mr.Iid) + "/award_emoji")
	if err != nil {
		return err
	}

	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return err
	}

	req.Header.Add("PRIVATE-TOKEN", gla.ApiKey)

	resp, err := gla.client.Do(req)
	response, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	var emojis []GitLabEmoji
	err = json.Unmarshal(response, &emojis)
	if err != nil {
		log.Fatal(err)
	}

	mr.Emojis = emojis

	return nil
}
