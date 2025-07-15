package twitterapi_reverse

import "time"

type SimpleTweet struct {
	TweetID      string     `json:"tweet_id"`
	Text         string     `json:"text"`
	CreatedAt    time.Time  `json:"created_at"`
	ReplyToID    *string    `json:"reply_to_id"`
	RepliesCount int        `json:"replies_count"`
	Author       SimpleUser `json:"author"`
}

type SimpleUser struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Name     string `json:"name"`
}

type TweetDetailResponse struct {
	Data struct {
		ThreadedConversationWithInjectionsV2 struct {
			Instructions []struct {
				Type    string `json:"type"`
				Entries []struct {
					EntryID   string `json:"entryId"`
					SortIndex string `json:"sortIndex"`
					Content   struct {
						EntryType   string `json:"entryType"`
						ItemContent struct {
							ItemType     string `json:"itemType"`
							TweetResults struct {
								Result struct {
									RestID string `json:"rest_id"`
									Core   struct {
										UserResults struct {
											Result struct {
												ID     string `json:"id"`
												RestID string `json:"rest_id"`
												Core   struct {
													ScreenName string `json:"screen_name"`
													Name       string `json:"name"`
												} `json:"core"`
											} `json:"result"`
										} `json:"user_results"`
									} `json:"core"`
									Legacy struct {
										FullText             string `json:"full_text"`
										CreatedAt            string `json:"created_at"`
										InReplyToStatusIDStr string `json:"in_reply_to_status_id_str"`
										ReplyCount           int    `json:"reply_count"`
									} `json:"legacy"`
								} `json:"result"`
							} `json:"tweet_results"`
						} `json:"itemContent"`
					} `json:"content"`
				} `json:"entries"`
			} `json:"instructions"`
		} `json:"threaded_conversation_with_injections_v2"`
	} `json:"data"`
}

type CommunityTweetsResponse struct {
	Data struct {
		CommunityResults struct {
			Result struct {
				RankedCommunityTimeline struct {
					Timeline struct {
						Instructions []struct {
							Type    string `json:"type"`
							Entries []struct {
								EntryID   string `json:"entryId"`
								SortIndex string `json:"sortIndex"`
								Content   struct {
									EntryType   string `json:"entryType"`
									ItemContent struct {
										ItemType     string `json:"itemType"`
										TweetResults struct {
											Result struct {
												RestID string `json:"rest_id"`
												Core   struct {
													UserResults struct {
														Result struct {
															ID     string `json:"id"`
															RestID string `json:"rest_id"`
															Core   struct {
																ScreenName string `json:"screen_name"`
																Name       string `json:"name"`
															} `json:"core"`
														} `json:"result"`
													} `json:"user_results"`
												} `json:"core"`
												Legacy struct {
													FullText             string `json:"full_text"`
													CreatedAt            string `json:"created_at"`
													InReplyToStatusIDStr string `json:"in_reply_to_status_id_str"`
													ReplyCount           int    `json:"reply_count"`
												} `json:"legacy"`
											} `json:"result"`
										} `json:"tweet_results"`
									} `json:"itemContent"`
								} `json:"content"`
							} `json:"entries"`
						} `json:"instructions"`
					} `json:"timeline"`
				} `json:"ranked_community_timeline"`
			} `json:"result"`
		} `json:"communityResults"`
	} `json:"data"`
}
