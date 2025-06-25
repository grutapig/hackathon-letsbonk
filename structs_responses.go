package main

type APIResponse struct {
	StatusCode int                 `json:"status_code"`
	Headers    map[string][]string `json:"headers"`
	RawBody    []byte              `json:"raw_body"`
}

type Author struct {
	Type           string `json:"type"`
	UserName       string `json:"userName"`
	Url            string `json:"url"`
	TwitterUrl     string `json:"twitterUrl"`
	Id             string `json:"id"`
	Name           string `json:"name"`
	ProfilePicture string `json:"profilePicture"`
	CoverPicture   string `json:"coverPicture"`
	Description    string `json:"description"`
	Location       string `json:"location"`
	Followers      int    `json:"followers"`
	Following      int    `json:"following"`
	Status         string `json:"status"`
	CanDm          bool   `json:"canDm"`
	CanMediaTag    bool   `json:"canMediaTag"`
	CreatedAt      string `json:"createdAt"`
	Entities       struct {
		Description struct {
			Symbols []struct {
				Indices []int  `json:"indices"`
				Text    string `json:"text"`
			} `json:"symbols,omitempty"`
			Urls []struct {
				DisplayUrl  string `json:"display_url"`
				ExpandedUrl string `json:"expanded_url"`
				Indices     []int  `json:"indices"`
				Url         string `json:"url"`
			} `json:"urls,omitempty"`
			UserMentions []struct {
				IdStr      string `json:"id_str"`
				Indices    []int  `json:"indices"`
				Name       string `json:"name"`
				ScreenName string `json:"screen_name"`
			} `json:"user_mentions,omitempty"`
			Hashtags []struct {
				Indices []int  `json:"indices"`
				Text    string `json:"text"`
			} `json:"hashtags,omitempty"`
		} `json:"description"`
		Url struct {
			Urls []struct {
				DisplayUrl  string `json:"display_url"`
				ExpandedUrl string `json:"expanded_url"`
				Indices     []int  `json:"indices"`
				Url         string `json:"url"`
			} `json:"urls"`
		} `json:"url,omitempty"`
	} `json:"entities"`
	FavouritesCount            int           `json:"favouritesCount"`
	HasCustomTimelines         bool          `json:"hasCustomTimelines"`
	IsTranslator               bool          `json:"isTranslator"`
	MediaCount                 int           `json:"mediaCount"`
	StatusesCount              int           `json:"statusesCount"`
	WithheldInCountries        []interface{} `json:"withheldInCountries"`
	AffiliatesHighlightedLabel struct {
	} `json:"affiliatesHighlightedLabel"`
	PossiblySensitive bool     `json:"possiblySensitive"`
	PinnedTweetIds    []string `json:"pinnedTweetIds"`
	ProfileBio        struct {
		Description string `json:"description"`
		Entities    struct {
			Description struct {
				Symbols []struct {
					Indices []int  `json:"indices"`
					Text    string `json:"text"`
				} `json:"symbols,omitempty"`
				Urls []struct {
					DisplayUrl  string `json:"display_url"`
					ExpandedUrl string `json:"expanded_url"`
					Indices     []int  `json:"indices"`
					Url         string `json:"url"`
				} `json:"urls,omitempty"`
				UserMentions []struct {
					IdStr      string `json:"id_str"`
					Indices    []int  `json:"indices"`
					Name       string `json:"name"`
					ScreenName string `json:"screen_name"`
				} `json:"user_mentions,omitempty"`
				Hashtags []struct {
					Indices []int  `json:"indices"`
					Text    string `json:"text"`
				} `json:"hashtags,omitempty"`
			} `json:"description"`
			Url struct {
				Urls []struct {
					DisplayUrl  string `json:"display_url"`
					ExpandedUrl string `json:"expanded_url"`
					Indices     []int  `json:"indices"`
					Url         string `json:"url"`
				} `json:"urls"`
			} `json:"url,omitempty"`
		} `json:"entities"`
		WithheldInCountries []interface{} `json:"withheld_in_countries"`
	} `json:"profile_bio"`
}

type Tweet struct {
	Type              string      `json:"type"`
	Id                string      `json:"id"`
	Url               string      `json:"url"`
	TwitterUrl        string      `json:"twitterUrl"`
	Text              string      `json:"text"`
	Source            string      `json:"source"`
	RetweetCount      int         `json:"retweetCount"`
	ReplyCount        int         `json:"replyCount"`
	LikeCount         int         `json:"likeCount"`
	QuoteCount        int         `json:"quoteCount"`
	ViewCount         int         `json:"viewCount"`
	CreatedAt         string      `json:"createdAt"`
	Lang              string      `json:"lang"`
	BookmarkCount     int         `json:"bookmarkCount"`
	IsReply           bool        `json:"isReply"`
	InReplyToId       interface{} `json:"inReplyToId"`
	ConversationId    string      `json:"conversationId"`
	InReplyToUserId   interface{} `json:"inReplyToUserId"`
	InReplyToUsername interface{} `json:"inReplyToUsername"`
	Author            Author      `json:"author"`
	ExtendedEntities  struct {
		Media []struct {
			AllowDownloadStatus struct {
				AllowDownload bool `json:"allow_download"`
			} `json:"allow_download_status,omitempty"`
			DisplayUrl           string `json:"display_url"`
			ExpandedUrl          string `json:"expanded_url"`
			ExtMediaAvailability struct {
				Status string `json:"status"`
			} `json:"ext_media_availability"`
			Features struct {
				Large struct {
					Faces []struct {
						H int `json:"h"`
						W int `json:"w"`
						X int `json:"x"`
						Y int `json:"y"`
					} `json:"faces,omitempty"`
				} `json:"large"`
				Orig struct {
					Faces []struct {
						H int `json:"h"`
						W int `json:"w"`
						X int `json:"x"`
						Y int `json:"y"`
					} `json:"faces,omitempty"`
				} `json:"orig"`
			} `json:"features,omitempty"`
			IdStr        string `json:"id_str"`
			Indices      []int  `json:"indices"`
			MediaKey     string `json:"media_key"`
			MediaResults struct {
				Id     string `json:"id"`
				Result struct {
					Typename string `json:"__typename"`
					Id       string `json:"id"`
					MediaKey string `json:"media_key"`
				} `json:"result"`
			} `json:"media_results"`
			MediaUrlHttps string `json:"media_url_https"`
			OriginalInfo  struct {
				FocusRects []struct {
					H int `json:"h"`
					W int `json:"w"`
					X int `json:"x"`
					Y int `json:"y"`
				} `json:"focus_rects"`
				Height int `json:"height"`
				Width  int `json:"width"`
			} `json:"original_info"`
			Sizes struct {
				Large struct {
					H int `json:"h"`
					W int `json:"w"`
				} `json:"large"`
			} `json:"sizes"`
			Type       string `json:"type"`
			Url        string `json:"url"`
			ExtAltText string `json:"ext_alt_text,omitempty"`
			VideoInfo  struct {
				AspectRatio []int `json:"aspect_ratio"`
				Variants    []struct {
					Bitrate     int    `json:"bitrate"`
					ContentType string `json:"content_type"`
					Url         string `json:"url"`
				} `json:"variants"`
			} `json:"video_info,omitempty"`
		} `json:"media,omitempty"`
	} `json:"extendedEntities"`
	Card  interface{} `json:"card"`
	Place struct {
	} `json:"place"`
	Entities struct {
		Symbols []struct {
			Indices []int  `json:"indices"`
			Text    string `json:"text"`
		} `json:"symbols,omitempty"`
		UserMentions []struct {
			IdStr      string `json:"id_str"`
			Indices    []int  `json:"indices"`
			Name       string `json:"name"`
			ScreenName string `json:"screen_name"`
		} `json:"user_mentions,omitempty"`
		Hashtags []struct {
			Indices []int  `json:"indices"`
			Text    string `json:"text"`
		} `json:"hashtags,omitempty"`
	} `json:"entities"`
}
type CommunityTweetsResponse struct {
	Tweets     []Tweet `json:"tweets"`
	HasNext    bool    `json:"has_next"`
	NextCursor string  `json:"next_cursor"`
	Status     string  `json:"status"`
	Msg        string  `json:"msg"`
}
type TweetRepliesResponse struct {
	Tweets      []Tweet `json:"tweets"`
	HasNextPage bool    `json:"has_next_page"`
	NextCursor  string  `json:"next_cursor"`
	Status      string  `json:"status"`
	Msg         string  `json:"msg"`
}
type UserLastTweetsResponse struct {
	Status string `json:"status"`
	Code   int    `json:"code"`
	Msg    string `json:"msg"`
	Data   struct {
		PinTweet interface{} `json:"pin_tweet"`
		Tweets   []Tweet     `json:"tweets"`
	} `json:"data"`
	HasNextPage bool   `json:"has_next_page"`
	NextCursor  string `json:"next_cursor"`
}
type User struct {
	Id                   string  `json:"id"`
	Name                 string  `json:"name"`
	ScreenName           string  `json:"screen_name"`
	UserName             string  `json:"userName"`
	Location             string  `json:"location"`
	Url                  *string `json:"url"`
	Description          string  `json:"description"`
	Email                *string `json:"email"`
	Protected            bool    `json:"protected"`
	Verified             bool    `json:"verified"`
	FollowersCount       int     `json:"followers_count"`
	FollowingCount       int     `json:"following_count"`
	FriendsCount         int     `json:"friends_count"`
	FavouritesCount      int     `json:"favourites_count"`
	StatusesCount        int     `json:"statuses_count"`
	MediaTweetsCount     int     `json:"media_tweets_count"`
	CreatedAt            string  `json:"created_at"`
	ProfileBannerUrl     *string `json:"profile_banner_url"`
	ProfileImageUrlHttps string  `json:"profile_image_url_https"`
	CanDm                bool    `json:"can_dm"`
}
type UserFollowersResponse struct {
	Followers   []User `json:"followers"`
	HasNextPage bool   `json:"has_next_page"`
	NextCursor  string `json:"next_cursor"`
	Status      string `json:"status"`
	Msg         string `json:"msg"`
	Code        int    `json:"code"`
}
type UserFollowingsResponse struct {
	Followings  []User `json:"followings"`
	HasNextPage bool   `json:"has_next_page"`
	NextCursor  string `json:"next_cursor"`
	Status      string `json:"status"`
	Msg         string `json:"msg"`
	Code        int    `json:"code"`
}
