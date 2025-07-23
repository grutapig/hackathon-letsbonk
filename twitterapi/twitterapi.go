package twitterapi

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

type TwitterAPIService struct {
	apiKey         string
	httpClient     *http.Client
	existingTweets map[string]bool
	tweetStates    map[string]*TweetState
	tweetMutex     sync.RWMutex
	baseUrl        string
}

func NewTwitterAPIService(apiKey string, baseUrl string, proxyDSN string) *TwitterAPIService {
	transport := &http.Transport{}
	if proxyDSN != "" {
		proxyURL, err := url.Parse(proxyDSN)
		if err != nil {
			panic(err)
		}

		transport = &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: false,
			},
		}
	}

	return &TwitterAPIService{
		apiKey:  apiKey,
		baseUrl: baseUrl,
		httpClient: &http.Client{
			Timeout:   10 * time.Second,
			Transport: transport,
		},
		existingTweets: make(map[string]bool),
		tweetStates:    make(map[string]*TweetState),
	}
}

func (s *TwitterAPIService) makeRequest(uri string, params map[string]string) (*APIResponse, error) {
	req, err := http.NewRequest("GET", uri, nil)
	if err != nil {
		return nil, fmt.Errorf("error create request: %w", err)
	}

	req.Header.Set("X-API-Key", s.apiKey)
	req.Header.Set("Content-Type", "application/json")

	q := req.URL.Query()
	for key, value := range params {
		if value != "" && key == "cursor" {
			unescape, _ := url.QueryUnescape(value)
			q.Add(key, unescape)
		} else if value != "" {
			q.Add(key, value)
		}
	}

	req.URL.RawQuery = q.Encode()
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error send request: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error read response: %w", err)
	}

	return &APIResponse{
		StatusCode: resp.StatusCode,
		Headers:    resp.Header,
		RawBody:    bodyBytes,
	}, nil
}

func (s *TwitterAPIService) GetCommunityTweets(req CommunityTweetsRequest) (*CommunityTweetsResponse, error) {
	uri := s.baseUrl + "/twitter/community/tweets"

	params := map[string]string{
		"community_id": req.CommunityID,
		"cursor":       req.Cursor,
	}

	response, err := s.makeRequest(uri, params)
	if err != nil {
		return nil, fmt.Errorf("error community messages: %w", err)
	}
	if response.StatusCode != 200 {
		return nil, fmt.Errorf("error community message, status non 200: %s", string(response.RawBody))
	}
	communityTweetsResponse := CommunityTweetsResponse{}
	err = json.Unmarshal(response.RawBody, &communityTweetsResponse)
	return &communityTweetsResponse, err
}

func (s *TwitterAPIService) GetUserLastTweets(req UserLastTweetsRequest) (*UserLastTweetsResponse, error) {
	uri := s.baseUrl + "/twitter/user/last_tweets"

	params := map[string]string{
		"userId":         req.UserId,
		"userName":       req.UserName,
		"cursor":         req.Cursor,
		"includeReplies": strconv.FormatBool(req.IncludeReplies),
	}

	response, err := s.makeRequest(uri, params)
	if err != nil {
		return nil, fmt.Errorf("error last user tweets: %w", err)
	}
	if response.StatusCode != 200 {
		return nil, fmt.Errorf("error last user tweets, status non 200: %s", string(response.RawBody))
	}
	userLastTweetsResponse := UserLastTweetsResponse{}
	err = json.Unmarshal(response.RawBody, &userLastTweetsResponse)
	return &userLastTweetsResponse, err
}

func (s *TwitterAPIService) GetTweetReplies(req TweetRepliesRequest) (*TweetRepliesResponse, error) {
	uri := s.baseUrl + "/twitter/tweet/replies"

	params := map[string]string{
		"tweetId": req.TweetID,
		"cursor":  req.Cursor,
	}
	if req.SinceTime > 0 {
		params["sinceTime"] = strconv.Itoa(int(req.SinceTime))
	}

	response, err := s.makeRequest(uri, params)
	if err != nil {
		return nil, fmt.Errorf("error last tweets: %w", err)
	}
	if response.StatusCode != 200 {
		return nil, fmt.Errorf("error last tweets, status non 200: %s", string(response.RawBody))
	}
	tweetRepliesResponse := TweetRepliesResponse{}
	err = json.Unmarshal(response.RawBody, &tweetRepliesResponse)
	return &tweetRepliesResponse, err
}
func (s *TwitterAPIService) GetTweetThreadContext(req TweetRepliesRequest) (*TweetRepliesResponse, error) {
	uri := s.baseUrl + "/twitter/tweet/thread_context"

	params := map[string]string{
		"tweetId": req.TweetID,
		"cursor":  req.Cursor,
	}
	if req.SinceTime > 0 {
		params["sinceTime"] = strconv.Itoa(int(req.SinceTime))
	}

	response, err := s.makeRequest(uri, params)
	if err != nil {
		return nil, fmt.Errorf("error last tweets: %w", err)
	}
	if response.StatusCode != 200 {
		return nil, fmt.Errorf("error last tweets, status non 200: %s", string(response.RawBody))
	}
	tweetRepliesResponse := TweetRepliesResponse{}
	err = json.Unmarshal(response.RawBody, &tweetRepliesResponse)
	return &tweetRepliesResponse, err
}
func (s *TwitterAPIService) GetUserFollowers(req UserFollowersRequest) (*UserFollowersResponse, error) {
	uri := s.baseUrl + "/twitter/user/followers"

	params := map[string]string{
		"userName": req.UserName,
		"cursor":   req.Cursor,
		"pageSize": strconv.Itoa(min(200, max(20, req.PageSize))),
	}

	response, err := s.makeRequest(uri, params)
	if err != nil {
		return nil, fmt.Errorf("error followers: %w", err)
	}
	if response.StatusCode != 200 {
		return nil, fmt.Errorf("error followers, status not 200: %s", string(response.RawBody))
	}
	userFollowersResponse := UserFollowersResponse{}

	err = json.Unmarshal(response.RawBody, &userFollowersResponse)
	return &userFollowersResponse, err
}
func (s *TwitterAPIService) GetUserFollowings(req UserFollowingsRequest) (*UserFollowingsResponse, error) {
	uri := s.baseUrl + "/twitter/user/followings"

	params := map[string]string{
		"userName": req.UserName,
		"cursor":   req.Cursor,
		"pageSize": strconv.Itoa(min(200, max(20, req.PageSize))),
	}

	response, err := s.makeRequest(uri, params)
	if err != nil {
		return nil, fmt.Errorf("error followings: %w", err)
	}
	if response.StatusCode != 200 {
		return nil, fmt.Errorf("error followings, status non 200: %s", string(response.RawBody))
	}
	userFollowingsResponse := UserFollowingsResponse{}
	err = json.Unmarshal(response.RawBody, &userFollowingsResponse)
	return &userFollowingsResponse, err
}

func (s *TwitterAPIService) GetTweetsByIds(tweetIds []string) (*TweetsByIdsResponse, error) {
	uri := s.baseUrl + "/twitter/tweets"

	params := map[string]string{
		"tweet_ids": strings.Join(tweetIds, ","),
	}

	response, err := s.makeRequest(uri, params)
	if err != nil {
		return nil, fmt.Errorf("error tweets_by_ids: %w", err)
	}
	if response.StatusCode != 200 {
		return nil, fmt.Errorf("error tweets_by_ids, status non 200: %s", string(response.RawBody))
	}
	tweetsByIdsResponse := TweetsByIdsResponse{}
	err = json.Unmarshal(response.RawBody, &tweetsByIdsResponse)
	return &tweetsByIdsResponse, err
}

func (s *TwitterAPIService) AdvancedSearch(request AdvancedSearchRequest) (*AdvancedSearchResponse, error) {
	uri := s.baseUrl + "/twitter/tweet/advanced_search"

	params := map[string]string{
		"query": request.Query,
	}

	if request.QueryType != "" {
		params["queryType"] = request.QueryType
	}

	if request.Cursor != "" {
		params["cursor"] = request.Cursor
	}

	response, err := s.makeRequest(uri, params)
	if err != nil {
		return nil, fmt.Errorf("error advanced_search: %w", err)
	}

	if response.StatusCode != 200 {
		return nil, fmt.Errorf("error advanced_search, status non 200: %s", string(response.RawBody))
	}

	searchResponse := AdvancedSearchResponse{}
	err = json.Unmarshal(response.RawBody, &searchResponse)
	return &searchResponse, err
}

func (s *TwitterAPIService) PostTweet(request PostTweetRequest) (*PostTweetResponse, error) {
	uri := s.baseUrl + "/twitter/create_tweet"
	requestBody, _ := json.Marshal(request)
	payload := bytes.NewReader(requestBody)

	req, _ := http.NewRequest("POST", uri, payload)
	req.Header.Set("X-API-Key", s.apiKey)
	req.Header.Set("Content-Type", "application/json")

	res, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error on post_tweet httpClient.Do: %s", err)
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("error on post_tweet io.ReadAll: %s", err)
	}

	if res.StatusCode != 200 {
		return nil, fmt.Errorf("error on post_tweet http status code is not 200: %d, body: %s", res.StatusCode, body)
	}
	fmt.Println(string(body))
	postTweetResponse := PostTweetResponse{}
	err = json.Unmarshal(body, &postTweetResponse)
	if len(postTweetResponse.Errors) > 0 {
		return &postTweetResponse, fmt.Errorf("error post tweet %s, error: %s", request.TweetText, postTweetResponse.Errors[0].Message)
	}
	return &postTweetResponse, err
}
