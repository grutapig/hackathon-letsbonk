package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"
)

const TWITTER_MESSAGE_TYPE_POST = "post"
const TWITTER_MESSAGE_TYPE_TWEET = "tweet"

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

func (s *TwitterAPIService) makeRequest(url string, params map[string]string) (*APIResponse, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error create request: %w", err)
	}

	req.Header.Set("X-API-Key", s.apiKey)
	req.Header.Set("Content-Type", "application/json")

	q := req.URL.Query()
	for key, value := range params {
		if value != "" {
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

func (s *TwitterAPIService) StartCommunityMonitoring(communityID string, pollInterval time.Duration) chan NewMessage {
	messageChan := make(chan NewMessage, 100)

	go func() {
		defer close(messageChan)

		for {
			log.Printf("checking again...")
			err := s.checkForNewMessages(communityID, messageChan)
			if err != nil {
				log.Printf("Error monitoring community %s: %v", communityID, err)
			}
			time.Sleep(pollInterval)
		}
	}()

	return messageChan
}

func (s *TwitterAPIService) checkForNewMessages(communityID string, messageChan chan NewMessage) error {
	communityResp, err := s.GetCommunityTweets(CommunityTweetsRequest{
		CommunityID: communityID,
		Cursor:      "",
	})
	if err != nil {
		return fmt.Errorf("failed to get community tweets: %w", err)
	}

	for i, tweet := range communityResp.Tweets {
		s.tweetMutex.Lock()
		_, exists := s.existingTweets[tweet.Id]
		previousState := s.tweetStates[tweet.Id]
		fmt.Println("! ", i, tweet.Text, tweet.ReplyCount)
		if !exists {
			s.existingTweets[tweet.Id] = true
			s.tweetStates[tweet.Id] = &TweetState{
				ID:         tweet.Id,
				ReplyCount: tweet.ReplyCount,
				LastCheck:  time.Now(),
				SinceTime:  time.Now(),
			}
			s.tweetMutex.Unlock()

			newMsg := NewMessage{
				MessageType: MessageTypeNewPost,
				TweetID:     tweet.Id,
				Author: struct {
					UserName string
					Name     string
					ID       string
				}{
					UserName: tweet.Author.UserName,
					Name:     tweet.Author.Name,
					ID:       tweet.Author.Id,
				},
				Text:         tweet.Text,
				CreatedAt:    tweet.CreatedAt,
				ReplyCount:   tweet.ReplyCount,
				LikeCount:    tweet.LikeCount,
				RetweetCount: tweet.RetweetCount,
			}

			select {
			case messageChan <- newMsg:
			default:
				log.Printf("Message channel full, dropping new post %s", tweet.Id)
			}
			fmt.Println("not exists first check:", tweet.Text, tweet.ReplyCount)
			go s.monitorTweetReplies(tweet, time.Unix(0, 0), messageChan)
		} else {
			if tweet.ReplyCount > previousState.ReplyCount {
				s.tweetStates[tweet.Id].ReplyCount = tweet.ReplyCount
				s.tweetStates[tweet.Id].LastCheck = time.Now()
				sinceTime := previousState.SinceTime
				s.tweetStates[tweet.Id].SinceTime = time.Now()
				s.tweetMutex.Unlock()
				fmt.Println("exists state: replies count more:", tweet.Text, tweet.ReplyCount, previousState.ReplyCount)
				go s.monitorTweetReplies(tweet, sinceTime, messageChan)
			} else {
				fmt.Println("same replies count:", tweet.Id, tweet.Text, "current count:", tweet.ReplyCount, "previous count:", previousState.ReplyCount)
				s.tweetMutex.Unlock()
			}
		}
	}

	return nil
}

func (s *TwitterAPIService) monitorTweetReplies(parentTweet Tweet, sinceTime time.Time, messageChan chan NewMessage) {
	repliesResp, err := s.GetTweetReplies(TweetRepliesRequest{
		TweetID:   parentTweet.Id,
		Cursor:    "",
		SinceTime: sinceTime.Unix(),
	})
	if err != nil {
		log.Printf("Error getting replies for tweet %s: %v", parentTweet.Id, err)
		return
	}
	fmt.Println("got replies for: ", parentTweet.Text, "replyCount: ", parentTweet.ReplyCount, "found:", len(repliesResp.Tweets))
	for _, tweet := range repliesResp.Tweets {
		s.tweetMutex.Lock()
		_, exists := s.existingTweets[tweet.Id]

		if !exists {
			s.existingTweets[tweet.Id] = true
			s.tweetMutex.Unlock()

			newReplyMsg := NewMessage{
				MessageType:  MessageTypeNewReply,
				TweetID:      parentTweet.Id,
				ReplyTweetID: tweet.Id,
				Author: struct {
					UserName string
					Name     string
					ID       string
				}{
					UserName: tweet.Author.UserName,
					Name:     tweet.Author.Name,
					ID:       tweet.Author.Id,
				},
				Text:      tweet.Text,
				CreatedAt: tweet.CreatedAt,
				ParentTweet: struct {
					ID     string
					Author string
					Text   string
				}{
					ID:     parentTweet.Id,
					Author: parentTweet.Author.UserName,
					Text:   parentTweet.Text,
				},
				ReplyCount:   tweet.ReplyCount,
				LikeCount:    tweet.LikeCount,
				RetweetCount: tweet.RetweetCount,
			}

			select {
			case messageChan <- newReplyMsg:
			default:
				log.Printf("Message channel full, dropping reply %s", tweet.Id)
			}
		} else {
			s.tweetMutex.Unlock()
		}
	}
}
