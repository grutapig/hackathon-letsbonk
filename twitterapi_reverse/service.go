package twitterapi_reverse

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type TwitterReverseService struct {
	auth      *TwitterAuth
	client    *http.Client
	proxyURL  string
	baseURL   string
	userAgent string
	debug     bool
}

func NewTwitterReverseApi(auth *TwitterAuth, proxyURL string, debug bool) *TwitterReverseService {
	service := &TwitterReverseService{
		auth:      auth,
		proxyURL:  proxyURL,
		baseURL:   "https://x.com",
		userAgent: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0.0.0 Safari/537.36",
		debug:     debug,
	}

	service.initHTTPClient()
	return service
}

func (s *TwitterReverseService) initHTTPClient() {
	s.client = &http.Client{
		Timeout: 30 * time.Second,
	}

	if s.proxyURL != "" {
		proxyURL, err := url.Parse(s.proxyURL)
		if err == nil {
			s.client.Transport = &http.Transport{
				Proxy: http.ProxyURL(proxyURL),
			}
		}
	}
}

func (s *TwitterReverseService) UpdateAuth(auth *TwitterAuth) {
	s.auth = auth
}

func (s *TwitterReverseService) makeRequest(method, endpoint string, params map[string]interface{}) ([]byte, error) {
	reqURL := s.baseURL + endpoint

	if method == "GET" && params != nil {
		values := url.Values{}
		for key, value := range params {
			switch v := value.(type) {
			case string:
				values.Add(key, v)
			case map[string]interface{}:
				jsonBytes, _ := json.Marshal(v)
				values.Add(key, string(jsonBytes))
			default:
				values.Add(key, fmt.Sprintf("%v", v))
			}
		}
		reqURL += "?" + values.Encode()
	}

	req, err := http.NewRequest(method, reqURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", s.userAgent)
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-twitter-active-user", "yes")
	req.Header.Set("x-twitter-auth-type", "OAuth2Session")
	req.Header.Set("x-twitter-client-language", "en")
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Site", "same-origin")

	if s.auth != nil {
		if s.auth.Authorization != "" {
			req.Header.Set("Authorization", s.auth.Authorization)
		}
		if s.auth.XCSRFToken != "" {
			req.Header.Set("x-csrf-token", s.auth.XCSRFToken)
		}
		if s.auth.Cookie != "" {
			req.Header.Set("Cookie", s.auth.Cookie)
		}
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if s.debug {
		fmt.Printf("=== DEBUG: HTTP Response ===\n")
		fmt.Printf("Status: %d %s\n", resp.StatusCode, resp.Status)
		fmt.Printf("URL: %s\n", reqURL)
		fmt.Printf("Response Body:\n%s\n", string(body))
		fmt.Printf("=== END DEBUG ===\n")
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP error: %d %s - Response: %s", resp.StatusCode, resp.Status, string(body))
	}

	return body, nil
}

func (s *TwitterReverseService) GetTweetDetail(tweetID string) (*SimpleTweet, error) {
	variables := map[string]interface{}{
		"focalTweetId":                           tweetID,
		"referrer":                               "community",
		"with_rux_injections":                    false,
		"rankingMode":                            "Relevance",
		"includePromotedContent":                 true,
		"withCommunity":                          true,
		"withQuickPromoteEligibilityTweetFields": true,
		"withBirdwatchNotes":                     true,
		"withVoice":                              true,
	}

	features := map[string]interface{}{
		"rweb_video_screen_enabled":                                               false,
		"payments_enabled":                                                        false,
		"profile_label_improvements_pcf_label_in_post_enabled":                    true,
		"rweb_tipjar_consumption_enabled":                                         true,
		"verified_phone_label_enabled":                                            false,
		"creator_subscriptions_tweet_preview_api_enabled":                         true,
		"responsive_web_graphql_timeline_navigation_enabled":                      true,
		"responsive_web_graphql_skip_user_profile_image_extensions_enabled":       false,
		"premium_content_api_read_enabled":                                        false,
		"communities_web_enable_tweet_community_results_fetch":                    true,
		"c9s_tweet_anatomy_moderator_badge_enabled":                               true,
		"responsive_web_grok_analyze_button_fetch_trends_enabled":                 false,
		"responsive_web_grok_analyze_post_followups_enabled":                      true,
		"responsive_web_jetfuel_frame":                                            false,
		"responsive_web_grok_share_attachment_enabled":                            true,
		"articles_preview_enabled":                                                true,
		"responsive_web_edit_tweet_api_enabled":                                   true,
		"graphql_is_translatable_rweb_tweet_is_translatable_enabled":              true,
		"view_counts_everywhere_api_enabled":                                      true,
		"longform_notetweets_consumption_enabled":                                 true,
		"responsive_web_twitter_article_tweet_consumption_enabled":                true,
		"tweet_awards_web_tipping_enabled":                                        false,
		"responsive_web_grok_show_grok_translated_post":                           false,
		"responsive_web_grok_analysis_button_from_backend":                        true,
		"creator_subscriptions_quote_tweet_preview_enabled":                       false,
		"freedom_of_speech_not_reach_fetch_enabled":                               true,
		"standardized_nudges_misinfo":                                             true,
		"tweet_with_visibility_results_prefer_gql_limited_actions_policy_enabled": true,
		"longform_notetweets_rich_text_read_enabled":                              true,
		"longform_notetweets_inline_media_enabled":                                true,
		"responsive_web_grok_image_annotation_enabled":                            true,
		"responsive_web_enhance_cards_enabled":                                    false,
	}

	fieldToggles := map[string]interface{}{
		"withArticleRichContentState": true,
		"withArticlePlainText":        false,
		"withGrokAnalyze":             false,
		"withDisallowedReplyControls": false,
	}

	params := map[string]interface{}{
		"variables":    variables,
		"features":     features,
		"fieldToggles": fieldToggles,
	}

	data, err := s.makeRequest("GET", "/i/api/graphql/-0WTL1e9Pij-JWAF5ztCCA/TweetDetail", params)
	if err != nil {
		return nil, err
	}

	tweets, err := ParseThreadedConversationTweets(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse tweet detail response: %v", err)
	}

	if len(tweets) == 0 {
		return nil, fmt.Errorf("no tweets found in response")
	}

	for _, tweet := range tweets {
		if tweet.ID == tweetID {
			return convertTweetToSimple(tweet), nil
		}
	}

	return convertTweetToSimple(tweets[0]), nil
}

func (s *TwitterReverseService) GetCommunityTweets(communityID string, count int) ([]SimpleTweet, error) {
	variables := map[string]interface{}{
		"communityId":     communityID,
		"count":           count,
		"displayLocation": "Community",
		"rankingMode":     "Recency",
		"withCommunity":   true,
	}

	features := map[string]interface{}{
		"rweb_video_screen_enabled":                                               false,
		"payments_enabled":                                                        false,
		"profile_label_improvements_pcf_label_in_post_enabled":                    true,
		"rweb_tipjar_consumption_enabled":                                         true,
		"verified_phone_label_enabled":                                            false,
		"creator_subscriptions_tweet_preview_api_enabled":                         true,
		"responsive_web_graphql_timeline_navigation_enabled":                      true,
		"responsive_web_graphql_skip_user_profile_image_extensions_enabled":       false,
		"premium_content_api_read_enabled":                                        false,
		"communities_web_enable_tweet_community_results_fetch":                    true,
		"c9s_tweet_anatomy_moderator_badge_enabled":                               true,
		"responsive_web_grok_analyze_button_fetch_trends_enabled":                 false,
		"responsive_web_grok_analyze_post_followups_enabled":                      true,
		"responsive_web_jetfuel_frame":                                            true,
		"responsive_web_grok_share_attachment_enabled":                            true,
		"articles_preview_enabled":                                                true,
		"responsive_web_edit_tweet_api_enabled":                                   true,
		"graphql_is_translatable_rweb_tweet_is_translatable_enabled":              true,
		"view_counts_everywhere_api_enabled":                                      true,
		"longform_notetweets_consumption_enabled":                                 true,
		"responsive_web_twitter_article_tweet_consumption_enabled":                true,
		"tweet_awards_web_tipping_enabled":                                        false,
		"responsive_web_grok_show_grok_translated_post":                           false,
		"responsive_web_grok_analysis_button_from_backend":                        true,
		"creator_subscriptions_quote_tweet_preview_enabled":                       false,
		"freedom_of_speech_not_reach_fetch_enabled":                               true,
		"standardized_nudges_misinfo":                                             true,
		"tweet_with_visibility_results_prefer_gql_limited_actions_policy_enabled": true,
		"longform_notetweets_rich_text_read_enabled":                              true,
		"longform_notetweets_inline_media_enabled":                                true,
		"responsive_web_grok_image_annotation_enabled":                            true,
		"responsive_web_grok_community_note_auto_translation_is_enabled":          false,
		"responsive_web_enhance_cards_enabled":                                    false,
	}

	params := map[string]interface{}{
		"variables": variables,
		"features":  features,
	}

	data, err := s.makeRequest("GET", "/i/api/graphql/f_muosmN8WvS9muD_kmwxA/CommunityTweetsTimeline", params)
	if err != nil {
		return nil, err
	}

	communityTweetsResponse := CommunityTweetsResponse{}
	err = json.Unmarshal(data, &communityTweetsResponse)
	if err != nil {
		return nil, fmt.Errorf("json unmarshal community tweets error. community: %s, err: %s", communityID, err)
	}
	var simpleTweets []SimpleTweet
	for _, instruction := range communityTweetsResponse.Data.CommunityResults.Result.RankedCommunityTimeline.Timeline.Instructions {
		if instruction.Entry.EntryId != "" {
			tweet := instruction.Entry
			date, err := ParseTwitterTime(tweet.Content.ItemContent.TweetResults.Result.Legacy.CreatedAt)
			if err != nil {
				date = time.Time{}
			}
			simpleTweet := SimpleTweet{
				TweetID:      tweet.Content.ItemContent.TweetResults.Result.RestId,
				Text:         tweet.Content.ItemContent.TweetResults.Result.Legacy.FullText,
				CreatedAt:    date,
				RepliesCount: tweet.Content.ItemContent.TweetResults.Result.Legacy.ReplyCount,
				Author: SimpleUser{
					ID:       tweet.Content.ItemContent.TweetResults.Result.Legacy.UserIdStr,
					Username: tweet.Content.ItemContent.TweetResults.Result.Core.UserResults.Result.Core.ScreenName,
					Name:     tweet.Content.ItemContent.TweetResults.Result.Core.UserResults.Result.Core.Name,
				},
			}
			simpleTweets = append(simpleTweets, simpleTweet)
		}
		for _, tweet := range instruction.Entries {
			date, err := ParseTwitterTime(tweet.Content.ItemContent.TweetResults.Result.Legacy.CreatedAt)
			if err != nil {
				date = time.Time{}
			}
			simpleTweet := SimpleTweet{
				TweetID:      tweet.Content.ItemContent.TweetResults.Result.RestId,
				Text:         tweet.Content.ItemContent.TweetResults.Result.Legacy.FullText,
				CreatedAt:    date,
				RepliesCount: tweet.Content.ItemContent.TweetResults.Result.Legacy.ReplyCount,
				Author: SimpleUser{
					ID:       tweet.Content.ItemContent.TweetResults.Result.Legacy.UserIdStr,
					Username: tweet.Content.ItemContent.TweetResults.Result.Core.UserResults.Result.Core.ScreenName,
					Name:     tweet.Content.ItemContent.TweetResults.Result.Core.UserResults.Result.Core.Name,
				},
			}
			if simpleTweet.TweetID != "" {
				simpleTweets = append(simpleTweets, simpleTweet)
			}
		}
	}

	return simpleTweets, nil
}

func convertTweetToSimple(tweet Tweet) *SimpleTweet {
	return &SimpleTweet{
		TweetID:      tweet.ID,
		Text:         tweet.FullText,
		CreatedAt:    tweet.CreatedAt,
		ReplyToID:    tweet.InReplyToStatusID,
		RepliesCount: int(tweet.ReplyCount),
		Author: SimpleUser{
			ID:       tweet.Author.ID,
			Username: tweet.Author.ScreenName,
			Name:     tweet.Author.Name,
		},
	}
}

func (s *TwitterReverseService) GetNotifications() (*NotificationsResponse, error) {
	ver := strconv.Itoa(int(time.Now().Unix()))
	body, err := s.makeRequest(http.MethodGet, "/i/api/graphql/Wa5HH91bSTqp3ZvBfTEtzQ/NotificationsTimeline?variables=%7B%22timeline_type%22%3A%22All%22%2C%22count%22%3A45%7D&features=%7B%22rweb_video_screen_enabled%22%3Afalse%2C%22payments_enabled%22%3Afalse%2C%22profile_label_improvements_pcf_label_in_post_enabled%22%3Atrue%2C%22rweb_tipjar_consumption_enabled%22%3Atrue%2C%22verified_phone_label_enabled%22%3Afalse%2C%22creator_subscriptions_tweet_preview_api_enabled%22%3Atrue%2C%22responsive_web_graphql_timeline_navigation_enabled%22%3Atrue%2C%22responsive_web_graphql_skip_user_profile_image_extensions_enabled%22%3Afalse%2C%22premium_content_api_read_enabled%22%3Afalse%2C%22communities_web_enable_tweet_community_results_fetch%22%3Atrue%2C%22c9s_tweet_anatomy_moderator_badge_enabled%22%3Atrue%2C%22responsive_web_grok_analyze_button_fetch_trends_enabled%22%3Afalse%2C%22responsive_web_grok_analyze_post_followups_enabled%22%3Atrue%2C%22responsive_web_jetfuel_frame%22%3Atrue%2C%22responsive_web_grok_share_attachment_enabled%22%3Atrue%2C%22articles_preview_enabled%22%3Atrue%2C%22responsive_web_edit_tweet_api_enabled%22%3Atrue%2C%22graphql_is_translatable_rweb_tweet_is_translatable_enabled%22%3Atrue%2C%22view_counts_everywhere_api_enabled%22%3Atrue%2C%22longform_notetweets_consumption_enabled%22%3Atrue%2C%22responsive_web_twitter_article_tweet_consumption_enabled%22%3Atrue%2C%22tweet_awards_web_tipping_enabled%22%3Afalse%2C%22responsive_web_grok_show_grok_translated_post%22%3Afalse%2C%22responsive_web_grok_analysis_button_from_backend%22%3Atrue%2C%22creator_subscriptions_quote_tweet_preview_enabled%22%3Afalse%2C%22freedom_of_speech_not_reach_fetch_enabled%22%3Atrue%2C%22standardized_nudges_misinfo%22%3Atrue%2C%22tweet_with_visibility_results_prefer_gql_limited_actions_policy_enabled%22%3Atrue%2C%22longform_notetweets_rich_text_read_enabled%22%3Atrue%2C%22longform_notetweets_inline_media_enabled%22%3Atrue%2C%22responsive_web_grok_image_annotation_enabled%22%3Atrue%2C%22responsive_web_grok_community_note_auto_translation_is_enabled%22%3Afalse%2C%22responsive_web_enhance_cards_enabled%22%3Afalse%7D&ver="+ver, nil)
	if err != nil {
		return nil, fmt.Errorf("error on make request GetNotifications: %s", err)
	}
	notificationsResponse := &NotificationsResponse{}
	err = json.Unmarshal(body, notificationsResponse)
	if err != nil {
		return nil, fmt.Errorf("error unmarshal GetNotifications: %s", err)
	}
	return notificationsResponse, err
}
func (s *TwitterReverseService) GetNotificationsSimple() ([]SimpleTweet, error) {
	tweets := []SimpleTweet{}
	notificationsResponse, err := s.GetNotifications()
	if err != nil {
		return nil, fmt.Errorf("GetNotificationsSimple error: %s", err)
	}
	for _, instruction := range notificationsResponse.Data.ViewerV2.UserResults.Result.NotificationTimeline.Timeline.Instructions {
		for _, entry := range instruction.Entries {
			if entry.Content.ItemContent.TweetResults.Result.Legacy.FullText != "" {
				timeConverted, err := ParseTwitterTime(entry.Content.ItemContent.TweetResults.Result.Legacy.CreatedAt)
				if err != nil {
					log.Println("error on parse date of tweet: ", err)
				}
				replyToStatusIdStr := entry.Content.ItemContent.TweetResults.Result.Legacy.InReplyToStatusIdStr
				if replyToStatusIdStr == "" {
					replyToStatusIdStr = entry.Content.ItemContent.TweetResults.Result.Legacy.QuotedStatusIdStr
				}
				tweets = append(tweets, SimpleTweet{
					TweetID:         entry.Content.ItemContent.TweetResults.Result.Legacy.IdStr,
					Text:            entry.Content.ItemContent.TweetResults.Result.Legacy.FullText,
					CreatedAt:       timeConverted,
					ReplyToID:       replyToStatusIdStr,
					ReplyToUsername: entry.Content.ItemContent.TweetResults.Result.Legacy.InReplyToScreenName,
					RepliesCount:    entry.Content.ItemContent.TweetResults.Result.Legacy.ReplyCount,
					Author: SimpleUser{
						ID:       entry.Content.ItemContent.TweetResults.Result.Legacy.UserIdStr,
						Username: entry.Content.ItemContent.TweetResults.Result.Core.UserResults.Result.Core.ScreenName,
						Name:     entry.Content.ItemContent.TweetResults.Result.Core.UserResults.Result.Core.Name,
					},
				})
			}
		}
	}
	return tweets, nil
}
