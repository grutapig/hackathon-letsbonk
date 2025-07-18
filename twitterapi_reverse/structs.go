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
				Typename                string `json:"__typename"`
				RankedCommunityTimeline struct {
					Timeline struct {
						Instructions []struct {
							Type  string `json:"type"`
							Entry struct {
								EntryId   string `json:"entryId"`
								SortIndex string `json:"sortIndex"`
								Content   struct {
									EntryType   string `json:"entryType"`
									Typename    string `json:"__typename"`
									ItemContent struct {
										ItemType     string `json:"itemType"`
										Typename     string `json:"__typename"`
										TweetResults struct {
											Result struct {
												Typename string `json:"__typename"`
												RestId   string `json:"rest_id"`
												Core     struct {
													UserResults struct {
														Result struct {
															Typename                   string `json:"__typename"`
															Id                         string `json:"id"`
															RestId                     string `json:"rest_id"`
															AffiliatesHighlightedLabel struct {
															} `json:"affiliates_highlighted_label"`
															Avatar struct {
																ImageUrl string `json:"image_url"`
															} `json:"avatar"`
															Core struct {
																CreatedAt  string `json:"created_at"`
																Name       string `json:"name"`
																ScreenName string `json:"screen_name"`
															} `json:"core"`
															DmPermissions struct {
																CanDm bool `json:"can_dm"`
															} `json:"dm_permissions"`
															HasGraduatedAccess bool `json:"has_graduated_access"`
															IsBlueVerified     bool `json:"is_blue_verified"`
															Legacy             struct {
																DefaultProfile      bool   `json:"default_profile"`
																DefaultProfileImage bool   `json:"default_profile_image"`
																Description         string `json:"description"`
																Entities            struct {
																	Description struct {
																		Urls []struct {
																			DisplayUrl  string `json:"display_url"`
																			ExpandedUrl string `json:"expanded_url"`
																			Url         string `json:"url"`
																			Indices     []int  `json:"indices"`
																		} `json:"urls"`
																	} `json:"description"`
																	Url struct {
																		Urls []struct {
																			DisplayUrl  string `json:"display_url"`
																			ExpandedUrl string `json:"expanded_url"`
																			Url         string `json:"url"`
																			Indices     []int  `json:"indices"`
																		} `json:"urls"`
																	} `json:"url"`
																} `json:"entities"`
																FastFollowersCount      int           `json:"fast_followers_count"`
																FavouritesCount         int           `json:"favourites_count"`
																FollowersCount          int           `json:"followers_count"`
																FriendsCount            int           `json:"friends_count"`
																HasCustomTimelines      bool          `json:"has_custom_timelines"`
																IsTranslator            bool          `json:"is_translator"`
																ListedCount             int           `json:"listed_count"`
																MediaCount              int           `json:"media_count"`
																NormalFollowersCount    int           `json:"normal_followers_count"`
																PinnedTweetIdsStr       []string      `json:"pinned_tweet_ids_str"`
																PossiblySensitive       bool          `json:"possibly_sensitive"`
																ProfileBannerUrl        string        `json:"profile_banner_url"`
																ProfileInterstitialType string        `json:"profile_interstitial_type"`
																StatusesCount           int           `json:"statuses_count"`
																TranslatorType          string        `json:"translator_type"`
																Url                     string        `json:"url"`
																WantRetweets            bool          `json:"want_retweets"`
																WithheldInCountries     []interface{} `json:"withheld_in_countries"`
															} `json:"legacy"`
															Location struct {
																Location string `json:"location"`
															} `json:"location"`
															MediaPermissions struct {
																CanMediaTag bool `json:"can_media_tag"`
															} `json:"media_permissions"`
															ParodyCommentaryFanLabel string `json:"parody_commentary_fan_label"`
															ProfileImageShape        string `json:"profile_image_shape"`
															Professional             struct {
																RestId           string `json:"rest_id"`
																ProfessionalType string `json:"professional_type"`
																Category         []struct {
																	Id       int    `json:"id"`
																	Name     string `json:"name"`
																	IconName string `json:"icon_name"`
																} `json:"category"`
															} `json:"professional"`
															Privacy struct {
																Protected bool `json:"protected"`
															} `json:"privacy"`
															RelationshipPerspectives struct {
																Following bool `json:"following"`
															} `json:"relationship_perspectives"`
															TipjarSettings struct {
																IsEnabled bool `json:"is_enabled"`
															} `json:"tipjar_settings"`
															SuperFollowEligible bool `json:"super_follow_eligible"`
															Verification        struct {
																Verified bool `json:"verified"`
															} `json:"verification"`
														} `json:"result"`
													} `json:"user_results"`
												} `json:"core"`
												UnmentionData struct {
												} `json:"unmention_data"`
												EditControl struct {
													EditTweetIds       []string `json:"edit_tweet_ids"`
													EditableUntilMsecs string   `json:"editable_until_msecs"`
													IsEditEligible     bool     `json:"is_edit_eligible"`
													EditsRemaining     string   `json:"edits_remaining"`
												} `json:"edit_control"`
												IsTranslatable bool `json:"is_translatable"`
												Views          struct {
													Count string `json:"count"`
													State string `json:"state"`
												} `json:"views"`
												Source             string `json:"source"`
												GrokAnalysisButton bool   `json:"grok_analysis_button"`
												CommunityResults   struct {
													Result struct {
														Typename           string `json:"__typename"`
														IdStr              string `json:"id_str"`
														ViewerRelationship struct {
															ModerationState struct {
																Typename string `json:"__typename"`
															} `json:"moderation_state"`
														} `json:"viewer_relationship"`
													} `json:"result"`
												} `json:"community_results"`
												CommunityRelationship struct {
													Id              string `json:"id"`
													RestId          string `json:"rest_id"`
													ModerationState struct {
													} `json:"moderation_state"`
													Actions struct {
														PinActionResult struct {
															Typename string `json:"__typename"`
														} `json:"pin_action_result"`
														UnpinActionResult struct {
															Typename string `json:"__typename"`
														} `json:"unpin_action_result"`
													} `json:"actions"`
												} `json:"community_relationship"`
												AuthorCommunityRelationship struct {
													CommunityResults struct {
														Result struct {
															Typename    string        `json:"__typename"`
															IdStr       string        `json:"id_str"`
															Name        string        `json:"name"`
															Description string        `json:"description"`
															CreatedAt   int64         `json:"created_at"`
															Question    string        `json:"question"`
															SearchTags  []interface{} `json:"search_tags"`
															IsNsfw      bool          `json:"is_nsfw"`
															Actions     struct {
																DeleteActionResult struct {
																	Typename string `json:"__typename"`
																	Reason   string `json:"reason"`
																} `json:"delete_action_result"`
																JoinActionResult struct {
																	Typename string `json:"__typename"`
																} `json:"join_action_result"`
																LeaveActionResult struct {
																	Typename string `json:"__typename"`
																	Reason   string `json:"reason"`
																	Message  string `json:"message"`
																} `json:"leave_action_result"`
																PinActionResult struct {
																	Typename string `json:"__typename"`
																} `json:"pin_action_result"`
															} `json:"actions"`
															AdminResults struct {
																Result struct {
																	Typename                   string `json:"__typename"`
																	Id                         string `json:"id"`
																	RestId                     string `json:"rest_id"`
																	AffiliatesHighlightedLabel struct {
																	} `json:"affiliates_highlighted_label"`
																	Avatar struct {
																		ImageUrl string `json:"image_url"`
																	} `json:"avatar"`
																	Core struct {
																		CreatedAt  string `json:"created_at"`
																		Name       string `json:"name"`
																		ScreenName string `json:"screen_name"`
																	} `json:"core"`
																	DmPermissions struct {
																		CanDm bool `json:"can_dm"`
																	} `json:"dm_permissions"`
																	HasGraduatedAccess bool `json:"has_graduated_access"`
																	IsBlueVerified     bool `json:"is_blue_verified"`
																	Legacy             struct {
																		DefaultProfile      bool   `json:"default_profile"`
																		DefaultProfileImage bool   `json:"default_profile_image"`
																		Description         string `json:"description"`
																		Entities            struct {
																			Description struct {
																				Urls []struct {
																					DisplayUrl  string `json:"display_url"`
																					ExpandedUrl string `json:"expanded_url"`
																					Url         string `json:"url"`
																					Indices     []int  `json:"indices"`
																				} `json:"urls"`
																			} `json:"description"`
																			Url struct {
																				Urls []struct {
																					DisplayUrl  string `json:"display_url"`
																					ExpandedUrl string `json:"expanded_url"`
																					Url         string `json:"url"`
																					Indices     []int  `json:"indices"`
																				} `json:"urls"`
																			} `json:"url"`
																		} `json:"entities"`
																		FastFollowersCount      int           `json:"fast_followers_count"`
																		FavouritesCount         int           `json:"favourites_count"`
																		FollowersCount          int           `json:"followers_count"`
																		FriendsCount            int           `json:"friends_count"`
																		HasCustomTimelines      bool          `json:"has_custom_timelines"`
																		IsTranslator            bool          `json:"is_translator"`
																		ListedCount             int           `json:"listed_count"`
																		MediaCount              int           `json:"media_count"`
																		NormalFollowersCount    int           `json:"normal_followers_count"`
																		PinnedTweetIdsStr       []string      `json:"pinned_tweet_ids_str"`
																		PossiblySensitive       bool          `json:"possibly_sensitive"`
																		ProfileBannerUrl        string        `json:"profile_banner_url"`
																		ProfileInterstitialType string        `json:"profile_interstitial_type"`
																		StatusesCount           int           `json:"statuses_count"`
																		TranslatorType          string        `json:"translator_type"`
																		Url                     string        `json:"url"`
																		WantRetweets            bool          `json:"want_retweets"`
																		WithheldInCountries     []interface{} `json:"withheld_in_countries"`
																	} `json:"legacy"`
																	Location struct {
																		Location string `json:"location"`
																	} `json:"location"`
																	MediaPermissions struct {
																		CanMediaTag bool `json:"can_media_tag"`
																	} `json:"media_permissions"`
																	ParodyCommentaryFanLabel string `json:"parody_commentary_fan_label"`
																	ProfileImageShape        string `json:"profile_image_shape"`
																	Professional             struct {
																		RestId           string `json:"rest_id"`
																		ProfessionalType string `json:"professional_type"`
																		Category         []struct {
																			Id       int    `json:"id"`
																			Name     string `json:"name"`
																			IconName string `json:"icon_name"`
																		} `json:"category"`
																	} `json:"professional"`
																	Privacy struct {
																		Protected bool `json:"protected"`
																	} `json:"privacy"`
																	RelationshipPerspectives struct {
																		Following bool `json:"following"`
																	} `json:"relationship_perspectives"`
																	TipjarSettings struct {
																		IsEnabled bool `json:"is_enabled"`
																	} `json:"tipjar_settings"`
																	SuperFollowEligible bool `json:"super_follow_eligible"`
																	Verification        struct {
																		Verified bool `json:"verified"`
																	} `json:"verification"`
																} `json:"result"`
															} `json:"admin_results"`
															CreatorResults struct {
																Result struct {
																	Typename                   string `json:"__typename"`
																	Id                         string `json:"id"`
																	RestId                     string `json:"rest_id"`
																	AffiliatesHighlightedLabel struct {
																	} `json:"affiliates_highlighted_label"`
																	Avatar struct {
																		ImageUrl string `json:"image_url"`
																	} `json:"avatar"`
																	Core struct {
																		CreatedAt  string `json:"created_at"`
																		Name       string `json:"name"`
																		ScreenName string `json:"screen_name"`
																	} `json:"core"`
																	DmPermissions struct {
																		CanDm bool `json:"can_dm"`
																	} `json:"dm_permissions"`
																	HasGraduatedAccess bool `json:"has_graduated_access"`
																	IsBlueVerified     bool `json:"is_blue_verified"`
																	Legacy             struct {
																		DefaultProfile      bool   `json:"default_profile"`
																		DefaultProfileImage bool   `json:"default_profile_image"`
																		Description         string `json:"description"`
																		Entities            struct {
																			Description struct {
																				Urls []struct {
																					DisplayUrl  string `json:"display_url"`
																					ExpandedUrl string `json:"expanded_url"`
																					Url         string `json:"url"`
																					Indices     []int  `json:"indices"`
																				} `json:"urls"`
																			} `json:"description"`
																			Url struct {
																				Urls []struct {
																					DisplayUrl  string `json:"display_url"`
																					ExpandedUrl string `json:"expanded_url"`
																					Url         string `json:"url"`
																					Indices     []int  `json:"indices"`
																				} `json:"urls"`
																			} `json:"url"`
																		} `json:"entities"`
																		FastFollowersCount      int           `json:"fast_followers_count"`
																		FavouritesCount         int           `json:"favourites_count"`
																		FollowersCount          int           `json:"followers_count"`
																		FriendsCount            int           `json:"friends_count"`
																		HasCustomTimelines      bool          `json:"has_custom_timelines"`
																		IsTranslator            bool          `json:"is_translator"`
																		ListedCount             int           `json:"listed_count"`
																		MediaCount              int           `json:"media_count"`
																		NormalFollowersCount    int           `json:"normal_followers_count"`
																		PinnedTweetIdsStr       []string      `json:"pinned_tweet_ids_str"`
																		PossiblySensitive       bool          `json:"possibly_sensitive"`
																		ProfileBannerUrl        string        `json:"profile_banner_url"`
																		ProfileInterstitialType string        `json:"profile_interstitial_type"`
																		StatusesCount           int           `json:"statuses_count"`
																		TranslatorType          string        `json:"translator_type"`
																		Url                     string        `json:"url"`
																		WantRetweets            bool          `json:"want_retweets"`
																		WithheldInCountries     []interface{} `json:"withheld_in_countries"`
																	} `json:"legacy"`
																	Location struct {
																		Location string `json:"location"`
																	} `json:"location"`
																	MediaPermissions struct {
																		CanMediaTag bool `json:"can_media_tag"`
																	} `json:"media_permissions"`
																	ParodyCommentaryFanLabel string `json:"parody_commentary_fan_label"`
																	ProfileImageShape        string `json:"profile_image_shape"`
																	Professional             struct {
																		RestId           string `json:"rest_id"`
																		ProfessionalType string `json:"professional_type"`
																		Category         []struct {
																			Id       int    `json:"id"`
																			Name     string `json:"name"`
																			IconName string `json:"icon_name"`
																		} `json:"category"`
																	} `json:"professional"`
																	Privacy struct {
																		Protected bool `json:"protected"`
																	} `json:"privacy"`
																	RelationshipPerspectives struct {
																		Following bool `json:"following"`
																	} `json:"relationship_perspectives"`
																	TipjarSettings struct {
																		IsEnabled bool `json:"is_enabled"`
																	} `json:"tipjar_settings"`
																	SuperFollowEligible bool `json:"super_follow_eligible"`
																	Verification        struct {
																		Verified bool `json:"verified"`
																	} `json:"verification"`
																} `json:"result"`
															} `json:"creator_results"`
															InvitesResult struct {
																Typename string `json:"__typename"`
																Reason   string `json:"reason"`
																Message  string `json:"message"`
															} `json:"invites_result"`
															JoinPolicy             string `json:"join_policy"`
															InvitesPolicy          string `json:"invites_policy"`
															IsPinned               bool   `json:"is_pinned"`
															MembersFacepileResults []struct {
																Result struct {
																	Typename                   string `json:"__typename"`
																	Id                         string `json:"id"`
																	RestId                     string `json:"rest_id"`
																	AffiliatesHighlightedLabel struct {
																	} `json:"affiliates_highlighted_label"`
																	Avatar struct {
																		ImageUrl string `json:"image_url"`
																	} `json:"avatar"`
																	Core struct {
																		CreatedAt  string `json:"created_at"`
																		Name       string `json:"name"`
																		ScreenName string `json:"screen_name"`
																	} `json:"core"`
																	DmPermissions struct {
																		CanDm bool `json:"can_dm"`
																	} `json:"dm_permissions"`
																	HasGraduatedAccess bool `json:"has_graduated_access"`
																	IsBlueVerified     bool `json:"is_blue_verified"`
																	Legacy             struct {
																		DefaultProfile      bool   `json:"default_profile"`
																		DefaultProfileImage bool   `json:"default_profile_image"`
																		Description         string `json:"description"`
																		Entities            struct {
																			Description struct {
																				Urls []struct {
																					DisplayUrl  string `json:"display_url"`
																					ExpandedUrl string `json:"expanded_url"`
																					Url         string `json:"url"`
																					Indices     []int  `json:"indices"`
																				} `json:"urls"`
																			} `json:"description"`
																			Url struct {
																				Urls []struct {
																					DisplayUrl  string `json:"display_url"`
																					ExpandedUrl string `json:"expanded_url"`
																					Url         string `json:"url"`
																					Indices     []int  `json:"indices"`
																				} `json:"urls"`
																			} `json:"url,omitempty"`
																		} `json:"entities"`
																		FastFollowersCount      int           `json:"fast_followers_count"`
																		FavouritesCount         int           `json:"favourites_count"`
																		FollowersCount          int           `json:"followers_count"`
																		FriendsCount            int           `json:"friends_count"`
																		HasCustomTimelines      bool          `json:"has_custom_timelines"`
																		IsTranslator            bool          `json:"is_translator"`
																		ListedCount             int           `json:"listed_count"`
																		MediaCount              int           `json:"media_count"`
																		NormalFollowersCount    int           `json:"normal_followers_count"`
																		PinnedTweetIdsStr       []string      `json:"pinned_tweet_ids_str"`
																		PossiblySensitive       bool          `json:"possibly_sensitive"`
																		ProfileBannerUrl        string        `json:"profile_banner_url,omitempty"`
																		ProfileInterstitialType string        `json:"profile_interstitial_type"`
																		StatusesCount           int           `json:"statuses_count"`
																		TranslatorType          string        `json:"translator_type"`
																		Url                     string        `json:"url,omitempty"`
																		WantRetweets            bool          `json:"want_retweets"`
																		WithheldInCountries     []interface{} `json:"withheld_in_countries"`
																	} `json:"legacy"`
																	Location struct {
																		Location string `json:"location"`
																	} `json:"location"`
																	MediaPermissions struct {
																		CanMediaTag bool `json:"can_media_tag"`
																	} `json:"media_permissions"`
																	ParodyCommentaryFanLabel string `json:"parody_commentary_fan_label"`
																	ProfileImageShape        string `json:"profile_image_shape"`
																	Professional             struct {
																		RestId           string `json:"rest_id"`
																		ProfessionalType string `json:"professional_type"`
																		Category         []struct {
																			Id       int    `json:"id"`
																			Name     string `json:"name"`
																			IconName string `json:"icon_name"`
																		} `json:"category"`
																	} `json:"professional,omitempty"`
																	Privacy struct {
																		Protected bool `json:"protected"`
																	} `json:"privacy"`
																	RelationshipPerspectives struct {
																		Following bool `json:"following"`
																	} `json:"relationship_perspectives"`
																	TipjarSettings struct {
																		IsEnabled bool `json:"is_enabled,omitempty"`
																	} `json:"tipjar_settings"`
																	SuperFollowEligible bool `json:"super_follow_eligible,omitempty"`
																	Verification        struct {
																		Verified bool `json:"verified"`
																	} `json:"verification"`
																} `json:"result"`
															} `json:"members_facepile_results"`
															ModeratorCount int    `json:"moderator_count"`
															MemberCount    int    `json:"member_count"`
															Role           string `json:"role"`
															Rules          []struct {
																RestId string `json:"rest_id"`
																Name   string `json:"name"`
															} `json:"rules"`
															CustomBannerMedia struct {
																MediaInfo struct {
																	ColorInfo struct {
																		Palette []struct {
																			Rgb struct {
																				Red   int `json:"red"`
																				Green int `json:"green"`
																				Blue  int `json:"blue"`
																			} `json:"rgb"`
																			Percentage float64 `json:"percentage"`
																		} `json:"palette"`
																	} `json:"color_info"`
																	OriginalImgUrl    string `json:"original_img_url"`
																	OriginalImgWidth  int    `json:"original_img_width"`
																	OriginalImgHeight int    `json:"original_img_height"`
																	SalientRect       struct {
																		Left   int `json:"left"`
																		Top    int `json:"top"`
																		Width  int `json:"width"`
																		Height int `json:"height"`
																	} `json:"salient_rect"`
																} `json:"media_info"`
															} `json:"custom_banner_media"`
															DefaultBannerMedia struct {
																MediaInfo struct {
																	ColorInfo struct {
																		Palette []struct {
																			Rgb struct {
																				Red   int `json:"red"`
																				Green int `json:"green"`
																				Blue  int `json:"blue"`
																			} `json:"rgb"`
																			Percentage float64 `json:"percentage"`
																		} `json:"palette"`
																	} `json:"color_info"`
																	OriginalImgUrl    string `json:"original_img_url"`
																	OriginalImgWidth  int    `json:"original_img_width"`
																	OriginalImgHeight int    `json:"original_img_height"`
																} `json:"media_info"`
															} `json:"default_banner_media"`
															ViewerRelationship struct {
																ModerationState struct {
																	Typename string `json:"__typename"`
																} `json:"moderation_state"`
															} `json:"viewer_relationship"`
															JoinRequestsResult struct {
																Typename string `json:"__typename"`
															} `json:"join_requests_result"`
														} `json:"result"`
													} `json:"community_results"`
													Role        string `json:"role"`
													UserResults struct {
														Result struct {
															Typename                   string `json:"__typename"`
															Id                         string `json:"id"`
															RestId                     string `json:"rest_id"`
															AffiliatesHighlightedLabel struct {
															} `json:"affiliates_highlighted_label"`
															Avatar struct {
																ImageUrl string `json:"image_url"`
															} `json:"avatar"`
															Core struct {
																CreatedAt  string `json:"created_at"`
																Name       string `json:"name"`
																ScreenName string `json:"screen_name"`
															} `json:"core"`
															DmPermissions struct {
																CanDm bool `json:"can_dm"`
															} `json:"dm_permissions"`
															HasGraduatedAccess bool `json:"has_graduated_access"`
															IsBlueVerified     bool `json:"is_blue_verified"`
															Legacy             struct {
																DefaultProfile      bool   `json:"default_profile"`
																DefaultProfileImage bool   `json:"default_profile_image"`
																Description         string `json:"description"`
																Entities            struct {
																	Description struct {
																		Urls []struct {
																			DisplayUrl  string `json:"display_url"`
																			ExpandedUrl string `json:"expanded_url"`
																			Url         string `json:"url"`
																			Indices     []int  `json:"indices"`
																		} `json:"urls"`
																	} `json:"description"`
																	Url struct {
																		Urls []struct {
																			DisplayUrl  string `json:"display_url"`
																			ExpandedUrl string `json:"expanded_url"`
																			Url         string `json:"url"`
																			Indices     []int  `json:"indices"`
																		} `json:"urls"`
																	} `json:"url"`
																} `json:"entities"`
																FastFollowersCount      int           `json:"fast_followers_count"`
																FavouritesCount         int           `json:"favourites_count"`
																FollowersCount          int           `json:"followers_count"`
																FriendsCount            int           `json:"friends_count"`
																HasCustomTimelines      bool          `json:"has_custom_timelines"`
																IsTranslator            bool          `json:"is_translator"`
																ListedCount             int           `json:"listed_count"`
																MediaCount              int           `json:"media_count"`
																NormalFollowersCount    int           `json:"normal_followers_count"`
																PinnedTweetIdsStr       []string      `json:"pinned_tweet_ids_str"`
																PossiblySensitive       bool          `json:"possibly_sensitive"`
																ProfileBannerUrl        string        `json:"profile_banner_url"`
																ProfileInterstitialType string        `json:"profile_interstitial_type"`
																StatusesCount           int           `json:"statuses_count"`
																TranslatorType          string        `json:"translator_type"`
																Url                     string        `json:"url"`
																WantRetweets            bool          `json:"want_retweets"`
																WithheldInCountries     []interface{} `json:"withheld_in_countries"`
															} `json:"legacy"`
															Location struct {
																Location string `json:"location"`
															} `json:"location"`
															MediaPermissions struct {
																CanMediaTag bool `json:"can_media_tag"`
															} `json:"media_permissions"`
															ParodyCommentaryFanLabel string `json:"parody_commentary_fan_label"`
															ProfileImageShape        string `json:"profile_image_shape"`
															Professional             struct {
																RestId           string `json:"rest_id"`
																ProfessionalType string `json:"professional_type"`
																Category         []struct {
																	Id       int    `json:"id"`
																	Name     string `json:"name"`
																	IconName string `json:"icon_name"`
																} `json:"category"`
															} `json:"professional"`
															Privacy struct {
																Protected bool `json:"protected"`
															} `json:"privacy"`
															RelationshipPerspectives struct {
																Following bool `json:"following"`
															} `json:"relationship_perspectives"`
															TipjarSettings struct {
																IsEnabled bool `json:"is_enabled"`
															} `json:"tipjar_settings"`
															SuperFollowEligible bool `json:"super_follow_eligible"`
															Verification        struct {
																Verified bool `json:"verified"`
															} `json:"verification"`
														} `json:"result"`
													} `json:"user_results"`
												} `json:"author_community_relationship"`
												Legacy struct {
													BookmarkCount     int    `json:"bookmark_count"`
													Bookmarked        bool   `json:"bookmarked"`
													CreatedAt         string `json:"created_at"`
													ConversationIdStr string `json:"conversation_id_str"`
													DisplayTextRange  []int  `json:"display_text_range"`
													Entities          struct {
														Hashtags     []interface{} `json:"hashtags"`
														Symbols      []interface{} `json:"symbols"`
														Timestamps   []interface{} `json:"timestamps"`
														Urls         []interface{} `json:"urls"`
														UserMentions []interface{} `json:"user_mentions"`
													} `json:"entities"`
													FavoriteCount int    `json:"favorite_count"`
													Favorited     bool   `json:"favorited"`
													FullText      string `json:"full_text"`
													IsQuoteStatus bool   `json:"is_quote_status"`
													Lang          string `json:"lang"`
													QuoteCount    int    `json:"quote_count"`
													ReplyCount    int    `json:"reply_count"`
													RetweetCount  int    `json:"retweet_count"`
													Retweeted     bool   `json:"retweeted"`
													UserIdStr     string `json:"user_id_str"`
													IdStr         string `json:"id_str"`
												} `json:"legacy"`
											} `json:"result"`
										} `json:"tweet_results"`
										TweetDisplayType string `json:"tweetDisplayType"`
										SocialContext    struct {
											Type        string `json:"type"`
											ContextType string `json:"contextType"`
											Text        string `json:"text"`
										} `json:"socialContext"`
									} `json:"itemContent"`
									ClientEventInfo struct {
										Component string `json:"component"`
										Element   string `json:"element"`
									} `json:"clientEventInfo"`
								} `json:"content"`
							} `json:"entry,omitempty"`
							Entries []struct {
								EntryId   string `json:"entryId"`
								SortIndex string `json:"sortIndex"`
								Content   struct {
									EntryType   string `json:"entryType"`
									Typename    string `json:"__typename"`
									ItemContent struct {
										ItemType     string `json:"itemType"`
										Typename     string `json:"__typename"`
										TweetResults struct {
											Result struct {
												Typename string `json:"__typename"`
												RestId   string `json:"rest_id"`
												Core     struct {
													UserResults struct {
														Result struct {
															Typename                   string `json:"__typename"`
															Id                         string `json:"id"`
															RestId                     string `json:"rest_id"`
															AffiliatesHighlightedLabel struct {
															} `json:"affiliates_highlighted_label"`
															Avatar struct {
																ImageUrl string `json:"image_url"`
															} `json:"avatar"`
															Core struct {
																CreatedAt  string `json:"created_at"`
																Name       string `json:"name"`
																ScreenName string `json:"screen_name"`
															} `json:"core"`
															DmPermissions struct {
																CanDm bool `json:"can_dm"`
															} `json:"dm_permissions"`
															HasGraduatedAccess bool `json:"has_graduated_access"`
															IsBlueVerified     bool `json:"is_blue_verified"`
															Legacy             struct {
																DefaultProfile      bool   `json:"default_profile"`
																DefaultProfileImage bool   `json:"default_profile_image"`
																Description         string `json:"description"`
																Entities            struct {
																	Description struct {
																		Urls []struct {
																			DisplayUrl  string `json:"display_url"`
																			ExpandedUrl string `json:"expanded_url"`
																			Url         string `json:"url"`
																			Indices     []int  `json:"indices"`
																		} `json:"urls"`
																	} `json:"description"`
																	Url struct {
																		Urls []struct {
																			DisplayUrl  string `json:"display_url"`
																			ExpandedUrl string `json:"expanded_url"`
																			Url         string `json:"url"`
																			Indices     []int  `json:"indices"`
																		} `json:"urls"`
																	} `json:"url,omitempty"`
																} `json:"entities"`
																FastFollowersCount      int           `json:"fast_followers_count"`
																FavouritesCount         int           `json:"favourites_count"`
																FollowersCount          int           `json:"followers_count"`
																FriendsCount            int           `json:"friends_count"`
																HasCustomTimelines      bool          `json:"has_custom_timelines"`
																IsTranslator            bool          `json:"is_translator"`
																ListedCount             int           `json:"listed_count"`
																MediaCount              int           `json:"media_count"`
																NormalFollowersCount    int           `json:"normal_followers_count"`
																PinnedTweetIdsStr       []string      `json:"pinned_tweet_ids_str"`
																PossiblySensitive       bool          `json:"possibly_sensitive"`
																ProfileBannerUrl        string        `json:"profile_banner_url,omitempty"`
																ProfileInterstitialType string        `json:"profile_interstitial_type"`
																StatusesCount           int           `json:"statuses_count"`
																TranslatorType          string        `json:"translator_type"`
																Url                     string        `json:"url,omitempty"`
																WantRetweets            bool          `json:"want_retweets"`
																WithheldInCountries     []interface{} `json:"withheld_in_countries"`
															} `json:"legacy"`
															Location struct {
																Location string `json:"location"`
															} `json:"location"`
															MediaPermissions struct {
																CanMediaTag bool `json:"can_media_tag"`
															} `json:"media_permissions"`
															ParodyCommentaryFanLabel string `json:"parody_commentary_fan_label"`
															ProfileImageShape        string `json:"profile_image_shape"`
															Professional             struct {
																RestId           string `json:"rest_id"`
																ProfessionalType string `json:"professional_type"`
																Category         []struct {
																	Id       int    `json:"id"`
																	Name     string `json:"name"`
																	IconName string `json:"icon_name"`
																} `json:"category"`
															} `json:"professional,omitempty"`
															Privacy struct {
																Protected bool `json:"protected"`
															} `json:"privacy"`
															RelationshipPerspectives struct {
																Following bool `json:"following"`
															} `json:"relationship_perspectives"`
															TipjarSettings struct {
																IsEnabled bool `json:"is_enabled,omitempty"`
															} `json:"tipjar_settings"`
															SuperFollowEligible bool `json:"super_follow_eligible,omitempty"`
															Verification        struct {
																Verified bool `json:"verified"`
															} `json:"verification"`
														} `json:"result"`
													} `json:"user_results"`
												} `json:"core"`
												UnmentionData struct {
												} `json:"unmention_data"`
												EditControl struct {
													EditTweetIds       []string `json:"edit_tweet_ids"`
													EditableUntilMsecs string   `json:"editable_until_msecs"`
													IsEditEligible     bool     `json:"is_edit_eligible"`
													EditsRemaining     string   `json:"edits_remaining"`
												} `json:"edit_control"`
												IsTranslatable bool `json:"is_translatable"`
												Views          struct {
													Count string `json:"count"`
													State string `json:"state"`
												} `json:"views"`
												Source    string `json:"source"`
												NoteTweet struct {
													IsExpandable     bool `json:"is_expandable"`
													NoteTweetResults struct {
														Result struct {
															Id        string `json:"id"`
															Text      string `json:"text"`
															EntitySet struct {
																Hashtags []interface{} `json:"hashtags"`
																Symbols  []struct {
																	Indices []int  `json:"indices"`
																	Text    string `json:"text"`
																} `json:"symbols"`
																Urls         []interface{} `json:"urls"`
																UserMentions []interface{} `json:"user_mentions"`
																Timestamps   []interface{} `json:"timestamps,omitempty"`
															} `json:"entity_set"`
															Richtext struct {
																RichtextTags []struct {
																	FromIndex     int      `json:"from_index"`
																	ToIndex       int      `json:"to_index"`
																	RichtextTypes []string `json:"richtext_types"`
																} `json:"richtext_tags"`
															} `json:"richtext"`
															Media struct {
																InlineMedia []interface{} `json:"inline_media"`
															} `json:"media,omitempty"`
														} `json:"result"`
													} `json:"note_tweet_results"`
												} `json:"note_tweet,omitempty"`
												GrokAnalysisButton bool `json:"grok_analysis_button"`
												CommunityResults   struct {
													Result struct {
														Typename           string `json:"__typename"`
														IdStr              string `json:"id_str"`
														ViewerRelationship struct {
															ModerationState struct {
																Typename string `json:"__typename"`
															} `json:"moderation_state"`
														} `json:"viewer_relationship"`
													} `json:"result"`
												} `json:"community_results"`
												CommunityRelationship struct {
													Id              string `json:"id"`
													RestId          string `json:"rest_id"`
													ModerationState struct {
													} `json:"moderation_state"`
													Actions struct {
														PinActionResult struct {
															Typename string `json:"__typename"`
														} `json:"pin_action_result"`
														UnpinActionResult struct {
															Typename string `json:"__typename"`
														} `json:"unpin_action_result"`
													} `json:"actions"`
												} `json:"community_relationship"`
												AuthorCommunityRelationship struct {
													CommunityResults struct {
														Result struct {
															Typename    string        `json:"__typename"`
															IdStr       string        `json:"id_str"`
															Name        string        `json:"name"`
															Description string        `json:"description"`
															CreatedAt   int64         `json:"created_at"`
															Question    string        `json:"question"`
															SearchTags  []interface{} `json:"search_tags"`
															IsNsfw      bool          `json:"is_nsfw"`
															Actions     struct {
																DeleteActionResult struct {
																	Typename string `json:"__typename"`
																	Reason   string `json:"reason"`
																} `json:"delete_action_result"`
																JoinActionResult struct {
																	Typename string `json:"__typename"`
																} `json:"join_action_result"`
																LeaveActionResult struct {
																	Typename string `json:"__typename"`
																	Reason   string `json:"reason"`
																	Message  string `json:"message"`
																} `json:"leave_action_result"`
																PinActionResult struct {
																	Typename string `json:"__typename"`
																} `json:"pin_action_result"`
															} `json:"actions"`
															AdminResults struct {
																Result struct {
																	Typename                   string `json:"__typename"`
																	Id                         string `json:"id"`
																	RestId                     string `json:"rest_id"`
																	AffiliatesHighlightedLabel struct {
																	} `json:"affiliates_highlighted_label"`
																	Avatar struct {
																		ImageUrl string `json:"image_url"`
																	} `json:"avatar"`
																	Core struct {
																		CreatedAt  string `json:"created_at"`
																		Name       string `json:"name"`
																		ScreenName string `json:"screen_name"`
																	} `json:"core"`
																	DmPermissions struct {
																		CanDm bool `json:"can_dm"`
																	} `json:"dm_permissions"`
																	HasGraduatedAccess bool `json:"has_graduated_access"`
																	IsBlueVerified     bool `json:"is_blue_verified"`
																	Legacy             struct {
																		DefaultProfile      bool   `json:"default_profile"`
																		DefaultProfileImage bool   `json:"default_profile_image"`
																		Description         string `json:"description"`
																		Entities            struct {
																			Description struct {
																				Urls []struct {
																					DisplayUrl  string `json:"display_url"`
																					ExpandedUrl string `json:"expanded_url"`
																					Url         string `json:"url"`
																					Indices     []int  `json:"indices"`
																				} `json:"urls"`
																			} `json:"description"`
																			Url struct {
																				Urls []struct {
																					DisplayUrl  string `json:"display_url"`
																					ExpandedUrl string `json:"expanded_url"`
																					Url         string `json:"url"`
																					Indices     []int  `json:"indices"`
																				} `json:"urls"`
																			} `json:"url"`
																		} `json:"entities"`
																		FastFollowersCount      int           `json:"fast_followers_count"`
																		FavouritesCount         int           `json:"favourites_count"`
																		FollowersCount          int           `json:"followers_count"`
																		FriendsCount            int           `json:"friends_count"`
																		HasCustomTimelines      bool          `json:"has_custom_timelines"`
																		IsTranslator            bool          `json:"is_translator"`
																		ListedCount             int           `json:"listed_count"`
																		MediaCount              int           `json:"media_count"`
																		NormalFollowersCount    int           `json:"normal_followers_count"`
																		PinnedTweetIdsStr       []string      `json:"pinned_tweet_ids_str"`
																		PossiblySensitive       bool          `json:"possibly_sensitive"`
																		ProfileBannerUrl        string        `json:"profile_banner_url"`
																		ProfileInterstitialType string        `json:"profile_interstitial_type"`
																		StatusesCount           int           `json:"statuses_count"`
																		TranslatorType          string        `json:"translator_type"`
																		Url                     string        `json:"url"`
																		WantRetweets            bool          `json:"want_retweets"`
																		WithheldInCountries     []interface{} `json:"withheld_in_countries"`
																	} `json:"legacy"`
																	Location struct {
																		Location string `json:"location"`
																	} `json:"location"`
																	MediaPermissions struct {
																		CanMediaTag bool `json:"can_media_tag"`
																	} `json:"media_permissions"`
																	ParodyCommentaryFanLabel string `json:"parody_commentary_fan_label"`
																	ProfileImageShape        string `json:"profile_image_shape"`
																	Professional             struct {
																		RestId           string `json:"rest_id"`
																		ProfessionalType string `json:"professional_type"`
																		Category         []struct {
																			Id       int    `json:"id"`
																			Name     string `json:"name"`
																			IconName string `json:"icon_name"`
																		} `json:"category"`
																	} `json:"professional"`
																	Privacy struct {
																		Protected bool `json:"protected"`
																	} `json:"privacy"`
																	RelationshipPerspectives struct {
																		Following bool `json:"following"`
																	} `json:"relationship_perspectives"`
																	TipjarSettings struct {
																		IsEnabled bool `json:"is_enabled"`
																	} `json:"tipjar_settings"`
																	SuperFollowEligible bool `json:"super_follow_eligible"`
																	Verification        struct {
																		Verified bool `json:"verified"`
																	} `json:"verification"`
																} `json:"result"`
															} `json:"admin_results"`
															CreatorResults struct {
																Result struct {
																	Typename                   string `json:"__typename"`
																	Id                         string `json:"id"`
																	RestId                     string `json:"rest_id"`
																	AffiliatesHighlightedLabel struct {
																	} `json:"affiliates_highlighted_label"`
																	Avatar struct {
																		ImageUrl string `json:"image_url"`
																	} `json:"avatar"`
																	Core struct {
																		CreatedAt  string `json:"created_at"`
																		Name       string `json:"name"`
																		ScreenName string `json:"screen_name"`
																	} `json:"core"`
																	DmPermissions struct {
																		CanDm bool `json:"can_dm"`
																	} `json:"dm_permissions"`
																	HasGraduatedAccess bool `json:"has_graduated_access"`
																	IsBlueVerified     bool `json:"is_blue_verified"`
																	Legacy             struct {
																		DefaultProfile      bool   `json:"default_profile"`
																		DefaultProfileImage bool   `json:"default_profile_image"`
																		Description         string `json:"description"`
																		Entities            struct {
																			Description struct {
																				Urls []struct {
																					DisplayUrl  string `json:"display_url"`
																					ExpandedUrl string `json:"expanded_url"`
																					Url         string `json:"url"`
																					Indices     []int  `json:"indices"`
																				} `json:"urls"`
																			} `json:"description"`
																			Url struct {
																				Urls []struct {
																					DisplayUrl  string `json:"display_url"`
																					ExpandedUrl string `json:"expanded_url"`
																					Url         string `json:"url"`
																					Indices     []int  `json:"indices"`
																				} `json:"urls"`
																			} `json:"url"`
																		} `json:"entities"`
																		FastFollowersCount      int           `json:"fast_followers_count"`
																		FavouritesCount         int           `json:"favourites_count"`
																		FollowersCount          int           `json:"followers_count"`
																		FriendsCount            int           `json:"friends_count"`
																		HasCustomTimelines      bool          `json:"has_custom_timelines"`
																		IsTranslator            bool          `json:"is_translator"`
																		ListedCount             int           `json:"listed_count"`
																		MediaCount              int           `json:"media_count"`
																		NormalFollowersCount    int           `json:"normal_followers_count"`
																		PinnedTweetIdsStr       []string      `json:"pinned_tweet_ids_str"`
																		PossiblySensitive       bool          `json:"possibly_sensitive"`
																		ProfileBannerUrl        string        `json:"profile_banner_url"`
																		ProfileInterstitialType string        `json:"profile_interstitial_type"`
																		StatusesCount           int           `json:"statuses_count"`
																		TranslatorType          string        `json:"translator_type"`
																		Url                     string        `json:"url"`
																		WantRetweets            bool          `json:"want_retweets"`
																		WithheldInCountries     []interface{} `json:"withheld_in_countries"`
																	} `json:"legacy"`
																	Location struct {
																		Location string `json:"location"`
																	} `json:"location"`
																	MediaPermissions struct {
																		CanMediaTag bool `json:"can_media_tag"`
																	} `json:"media_permissions"`
																	ParodyCommentaryFanLabel string `json:"parody_commentary_fan_label"`
																	ProfileImageShape        string `json:"profile_image_shape"`
																	Professional             struct {
																		RestId           string `json:"rest_id"`
																		ProfessionalType string `json:"professional_type"`
																		Category         []struct {
																			Id       int    `json:"id"`
																			Name     string `json:"name"`
																			IconName string `json:"icon_name"`
																		} `json:"category"`
																	} `json:"professional"`
																	Privacy struct {
																		Protected bool `json:"protected"`
																	} `json:"privacy"`
																	RelationshipPerspectives struct {
																		Following bool `json:"following"`
																	} `json:"relationship_perspectives"`
																	TipjarSettings struct {
																		IsEnabled bool `json:"is_enabled"`
																	} `json:"tipjar_settings"`
																	SuperFollowEligible bool `json:"super_follow_eligible"`
																	Verification        struct {
																		Verified bool `json:"verified"`
																	} `json:"verification"`
																} `json:"result"`
															} `json:"creator_results"`
															InvitesResult struct {
																Typename string `json:"__typename"`
																Reason   string `json:"reason"`
																Message  string `json:"message"`
															} `json:"invites_result"`
															JoinPolicy             string `json:"join_policy"`
															InvitesPolicy          string `json:"invites_policy"`
															IsPinned               bool   `json:"is_pinned"`
															MembersFacepileResults []struct {
																Result struct {
																	Typename                   string `json:"__typename"`
																	Id                         string `json:"id"`
																	RestId                     string `json:"rest_id"`
																	AffiliatesHighlightedLabel struct {
																	} `json:"affiliates_highlighted_label"`
																	Avatar struct {
																		ImageUrl string `json:"image_url"`
																	} `json:"avatar"`
																	Core struct {
																		CreatedAt  string `json:"created_at"`
																		Name       string `json:"name"`
																		ScreenName string `json:"screen_name"`
																	} `json:"core"`
																	DmPermissions struct {
																		CanDm bool `json:"can_dm"`
																	} `json:"dm_permissions"`
																	HasGraduatedAccess bool `json:"has_graduated_access"`
																	IsBlueVerified     bool `json:"is_blue_verified"`
																	Legacy             struct {
																		DefaultProfile      bool   `json:"default_profile"`
																		DefaultProfileImage bool   `json:"default_profile_image"`
																		Description         string `json:"description"`
																		Entities            struct {
																			Description struct {
																				Urls []struct {
																					DisplayUrl  string `json:"display_url"`
																					ExpandedUrl string `json:"expanded_url"`
																					Url         string `json:"url"`
																					Indices     []int  `json:"indices"`
																				} `json:"urls"`
																			} `json:"description"`
																			Url struct {
																				Urls []struct {
																					DisplayUrl  string `json:"display_url"`
																					ExpandedUrl string `json:"expanded_url"`
																					Url         string `json:"url"`
																					Indices     []int  `json:"indices"`
																				} `json:"urls"`
																			} `json:"url,omitempty"`
																		} `json:"entities"`
																		FastFollowersCount      int           `json:"fast_followers_count"`
																		FavouritesCount         int           `json:"favourites_count"`
																		FollowersCount          int           `json:"followers_count"`
																		FriendsCount            int           `json:"friends_count"`
																		HasCustomTimelines      bool          `json:"has_custom_timelines"`
																		IsTranslator            bool          `json:"is_translator"`
																		ListedCount             int           `json:"listed_count"`
																		MediaCount              int           `json:"media_count"`
																		NormalFollowersCount    int           `json:"normal_followers_count"`
																		PinnedTweetIdsStr       []string      `json:"pinned_tweet_ids_str"`
																		PossiblySensitive       bool          `json:"possibly_sensitive"`
																		ProfileBannerUrl        string        `json:"profile_banner_url,omitempty"`
																		ProfileInterstitialType string        `json:"profile_interstitial_type"`
																		StatusesCount           int           `json:"statuses_count"`
																		TranslatorType          string        `json:"translator_type"`
																		Url                     string        `json:"url,omitempty"`
																		WantRetweets            bool          `json:"want_retweets"`
																		WithheldInCountries     []interface{} `json:"withheld_in_countries"`
																	} `json:"legacy"`
																	Location struct {
																		Location string `json:"location"`
																	} `json:"location"`
																	MediaPermissions struct {
																		CanMediaTag bool `json:"can_media_tag"`
																	} `json:"media_permissions"`
																	ParodyCommentaryFanLabel string `json:"parody_commentary_fan_label"`
																	ProfileImageShape        string `json:"profile_image_shape"`
																	Professional             struct {
																		RestId           string `json:"rest_id"`
																		ProfessionalType string `json:"professional_type"`
																		Category         []struct {
																			Id       int    `json:"id"`
																			Name     string `json:"name"`
																			IconName string `json:"icon_name"`
																		} `json:"category"`
																	} `json:"professional,omitempty"`
																	Privacy struct {
																		Protected bool `json:"protected"`
																	} `json:"privacy"`
																	RelationshipPerspectives struct {
																		Following bool `json:"following"`
																	} `json:"relationship_perspectives"`
																	TipjarSettings struct {
																		IsEnabled bool `json:"is_enabled,omitempty"`
																	} `json:"tipjar_settings"`
																	SuperFollowEligible bool `json:"super_follow_eligible,omitempty"`
																	Verification        struct {
																		Verified bool `json:"verified"`
																	} `json:"verification"`
																} `json:"result"`
															} `json:"members_facepile_results"`
															ModeratorCount int    `json:"moderator_count"`
															MemberCount    int    `json:"member_count"`
															Role           string `json:"role"`
															Rules          []struct {
																RestId string `json:"rest_id"`
																Name   string `json:"name"`
															} `json:"rules"`
															CustomBannerMedia struct {
																MediaInfo struct {
																	ColorInfo struct {
																		Palette []struct {
																			Rgb struct {
																				Red   int `json:"red"`
																				Green int `json:"green"`
																				Blue  int `json:"blue"`
																			} `json:"rgb"`
																			Percentage float64 `json:"percentage"`
																		} `json:"palette"`
																	} `json:"color_info"`
																	OriginalImgUrl    string `json:"original_img_url"`
																	OriginalImgWidth  int    `json:"original_img_width"`
																	OriginalImgHeight int    `json:"original_img_height"`
																	SalientRect       struct {
																		Left   int `json:"left"`
																		Top    int `json:"top"`
																		Width  int `json:"width"`
																		Height int `json:"height"`
																	} `json:"salient_rect"`
																} `json:"media_info"`
															} `json:"custom_banner_media"`
															DefaultBannerMedia struct {
																MediaInfo struct {
																	ColorInfo struct {
																		Palette []struct {
																			Rgb struct {
																				Red   int `json:"red"`
																				Green int `json:"green"`
																				Blue  int `json:"blue"`
																			} `json:"rgb"`
																			Percentage float64 `json:"percentage"`
																		} `json:"palette"`
																	} `json:"color_info"`
																	OriginalImgUrl    string `json:"original_img_url"`
																	OriginalImgWidth  int    `json:"original_img_width"`
																	OriginalImgHeight int    `json:"original_img_height"`
																} `json:"media_info"`
															} `json:"default_banner_media"`
															ViewerRelationship struct {
																ModerationState struct {
																	Typename string `json:"__typename"`
																} `json:"moderation_state"`
															} `json:"viewer_relationship"`
															JoinRequestsResult struct {
																Typename string `json:"__typename"`
															} `json:"join_requests_result"`
														} `json:"result"`
													} `json:"community_results"`
													Role        string `json:"role"`
													UserResults struct {
														Result struct {
															Typename                   string `json:"__typename"`
															Id                         string `json:"id"`
															RestId                     string `json:"rest_id"`
															AffiliatesHighlightedLabel struct {
															} `json:"affiliates_highlighted_label"`
															Avatar struct {
																ImageUrl string `json:"image_url"`
															} `json:"avatar"`
															Core struct {
																CreatedAt  string `json:"created_at"`
																Name       string `json:"name"`
																ScreenName string `json:"screen_name"`
															} `json:"core"`
															DmPermissions struct {
																CanDm bool `json:"can_dm"`
															} `json:"dm_permissions"`
															HasGraduatedAccess bool `json:"has_graduated_access"`
															IsBlueVerified     bool `json:"is_blue_verified"`
															Legacy             struct {
																DefaultProfile      bool   `json:"default_profile"`
																DefaultProfileImage bool   `json:"default_profile_image"`
																Description         string `json:"description"`
																Entities            struct {
																	Description struct {
																		Urls []struct {
																			DisplayUrl  string `json:"display_url"`
																			ExpandedUrl string `json:"expanded_url"`
																			Url         string `json:"url"`
																			Indices     []int  `json:"indices"`
																		} `json:"urls"`
																	} `json:"description"`
																	Url struct {
																		Urls []struct {
																			DisplayUrl  string `json:"display_url"`
																			ExpandedUrl string `json:"expanded_url"`
																			Url         string `json:"url"`
																			Indices     []int  `json:"indices"`
																		} `json:"urls"`
																	} `json:"url,omitempty"`
																} `json:"entities"`
																FastFollowersCount      int           `json:"fast_followers_count"`
																FavouritesCount         int           `json:"favourites_count"`
																FollowersCount          int           `json:"followers_count"`
																FriendsCount            int           `json:"friends_count"`
																HasCustomTimelines      bool          `json:"has_custom_timelines"`
																IsTranslator            bool          `json:"is_translator"`
																ListedCount             int           `json:"listed_count"`
																MediaCount              int           `json:"media_count"`
																NormalFollowersCount    int           `json:"normal_followers_count"`
																PinnedTweetIdsStr       []string      `json:"pinned_tweet_ids_str"`
																PossiblySensitive       bool          `json:"possibly_sensitive"`
																ProfileBannerUrl        string        `json:"profile_banner_url,omitempty"`
																ProfileInterstitialType string        `json:"profile_interstitial_type"`
																StatusesCount           int           `json:"statuses_count"`
																TranslatorType          string        `json:"translator_type"`
																Url                     string        `json:"url,omitempty"`
																WantRetweets            bool          `json:"want_retweets"`
																WithheldInCountries     []interface{} `json:"withheld_in_countries"`
															} `json:"legacy"`
															Location struct {
																Location string `json:"location"`
															} `json:"location"`
															MediaPermissions struct {
																CanMediaTag bool `json:"can_media_tag"`
															} `json:"media_permissions"`
															ParodyCommentaryFanLabel string `json:"parody_commentary_fan_label"`
															ProfileImageShape        string `json:"profile_image_shape"`
															Professional             struct {
																RestId           string `json:"rest_id"`
																ProfessionalType string `json:"professional_type"`
																Category         []struct {
																	Id       int    `json:"id"`
																	Name     string `json:"name"`
																	IconName string `json:"icon_name"`
																} `json:"category"`
															} `json:"professional,omitempty"`
															Privacy struct {
																Protected bool `json:"protected"`
															} `json:"privacy"`
															RelationshipPerspectives struct {
																Following bool `json:"following"`
															} `json:"relationship_perspectives"`
															TipjarSettings struct {
																IsEnabled bool `json:"is_enabled,omitempty"`
															} `json:"tipjar_settings"`
															SuperFollowEligible bool `json:"super_follow_eligible,omitempty"`
															Verification        struct {
																Verified bool `json:"verified"`
															} `json:"verification"`
														} `json:"result"`
													} `json:"user_results"`
												} `json:"author_community_relationship"`
												Legacy struct {
													BookmarkCount     int    `json:"bookmark_count"`
													Bookmarked        bool   `json:"bookmarked"`
													CreatedAt         string `json:"created_at"`
													ConversationIdStr string `json:"conversation_id_str"`
													DisplayTextRange  []int  `json:"display_text_range"`
													Entities          struct {
														Hashtags []struct {
															Indices []int  `json:"indices"`
															Text    string `json:"text"`
														} `json:"hashtags"`
														Symbols []struct {
															Indices []int  `json:"indices"`
															Text    string `json:"text"`
														} `json:"symbols"`
														Timestamps   []interface{} `json:"timestamps"`
														Urls         []interface{} `json:"urls"`
														UserMentions []struct {
															IdStr      string `json:"id_str"`
															Name       string `json:"name"`
															ScreenName string `json:"screen_name"`
															Indices    []int  `json:"indices"`
														} `json:"user_mentions"`
														Media []struct {
															DisplayUrl           string `json:"display_url"`
															ExpandedUrl          string `json:"expanded_url"`
															IdStr                string `json:"id_str"`
															Indices              []int  `json:"indices"`
															MediaKey             string `json:"media_key"`
															MediaUrlHttps        string `json:"media_url_https"`
															Type                 string `json:"type"`
															Url                  string `json:"url"`
															ExtMediaAvailability struct {
																Status string `json:"status"`
															} `json:"ext_media_availability"`
															Features struct {
																Large struct {
																	Faces []struct {
																		X int `json:"x"`
																		Y int `json:"y"`
																		H int `json:"h"`
																		W int `json:"w"`
																	} `json:"faces"`
																} `json:"large"`
																Medium struct {
																	Faces []struct {
																		X int `json:"x"`
																		Y int `json:"y"`
																		H int `json:"h"`
																		W int `json:"w"`
																	} `json:"faces"`
																} `json:"medium"`
																Small struct {
																	Faces []struct {
																		X int `json:"x"`
																		Y int `json:"y"`
																		H int `json:"h"`
																		W int `json:"w"`
																	} `json:"faces"`
																} `json:"small"`
																Orig struct {
																	Faces []struct {
																		X int `json:"x"`
																		Y int `json:"y"`
																		H int `json:"h"`
																		W int `json:"w"`
																	} `json:"faces"`
																} `json:"orig"`
																All struct {
																	Tags []struct {
																		UserId     string `json:"user_id"`
																		Name       string `json:"name"`
																		ScreenName string `json:"screen_name"`
																		Type       string `json:"type"`
																	} `json:"tags"`
																} `json:"all,omitempty"`
															} `json:"features,omitempty"`
															Sizes struct {
																Large struct {
																	H      int    `json:"h"`
																	W      int    `json:"w"`
																	Resize string `json:"resize"`
																} `json:"large"`
																Medium struct {
																	H      int    `json:"h"`
																	W      int    `json:"w"`
																	Resize string `json:"resize"`
																} `json:"medium"`
																Small struct {
																	H      int    `json:"h"`
																	W      int    `json:"w"`
																	Resize string `json:"resize"`
																} `json:"small"`
																Thumb struct {
																	H      int    `json:"h"`
																	W      int    `json:"w"`
																	Resize string `json:"resize"`
																} `json:"thumb"`
															} `json:"sizes"`
															OriginalInfo struct {
																Height     int `json:"height"`
																Width      int `json:"width"`
																FocusRects []struct {
																	X int `json:"x"`
																	Y int `json:"y"`
																	W int `json:"w"`
																	H int `json:"h"`
																} `json:"focus_rects"`
															} `json:"original_info"`
															MediaResults struct {
																Result struct {
																	MediaKey string `json:"media_key"`
																} `json:"result"`
															} `json:"media_results"`
															AdditionalMediaInfo struct {
																Monetizable bool `json:"monetizable"`
															} `json:"additional_media_info,omitempty"`
															AllowDownloadStatus struct {
																AllowDownload bool `json:"allow_download"`
															} `json:"allow_download_status,omitempty"`
															VideoInfo struct {
																AspectRatio    []int `json:"aspect_ratio"`
																DurationMillis int   `json:"duration_millis"`
																Variants       []struct {
																	ContentType string `json:"content_type"`
																	Url         string `json:"url"`
																	Bitrate     int    `json:"bitrate,omitempty"`
																} `json:"variants"`
															} `json:"video_info,omitempty"`
														} `json:"media,omitempty"`
													} `json:"entities"`
													FavoriteCount    int    `json:"favorite_count"`
													Favorited        bool   `json:"favorited"`
													FullText         string `json:"full_text"`
													IsQuoteStatus    bool   `json:"is_quote_status"`
													Lang             string `json:"lang"`
													QuoteCount       int    `json:"quote_count"`
													ReplyCount       int    `json:"reply_count"`
													RetweetCount     int    `json:"retweet_count"`
													Retweeted        bool   `json:"retweeted"`
													UserIdStr        string `json:"user_id_str"`
													IdStr            string `json:"id_str"`
													ExtendedEntities struct {
														Media []struct {
															DisplayUrl           string `json:"display_url"`
															ExpandedUrl          string `json:"expanded_url"`
															IdStr                string `json:"id_str"`
															Indices              []int  `json:"indices"`
															MediaKey             string `json:"media_key"`
															MediaUrlHttps        string `json:"media_url_https"`
															Type                 string `json:"type"`
															Url                  string `json:"url"`
															ExtMediaAvailability struct {
																Status string `json:"status"`
															} `json:"ext_media_availability"`
															Features struct {
																Large struct {
																	Faces []struct {
																		X int `json:"x"`
																		Y int `json:"y"`
																		H int `json:"h"`
																		W int `json:"w"`
																	} `json:"faces"`
																} `json:"large"`
																Medium struct {
																	Faces []struct {
																		X int `json:"x"`
																		Y int `json:"y"`
																		H int `json:"h"`
																		W int `json:"w"`
																	} `json:"faces"`
																} `json:"medium"`
																Small struct {
																	Faces []struct {
																		X int `json:"x"`
																		Y int `json:"y"`
																		H int `json:"h"`
																		W int `json:"w"`
																	} `json:"faces"`
																} `json:"small"`
																Orig struct {
																	Faces []struct {
																		X int `json:"x"`
																		Y int `json:"y"`
																		H int `json:"h"`
																		W int `json:"w"`
																	} `json:"faces"`
																} `json:"orig"`
																All struct {
																	Tags []struct {
																		UserId     string `json:"user_id"`
																		Name       string `json:"name"`
																		ScreenName string `json:"screen_name"`
																		Type       string `json:"type"`
																	} `json:"tags"`
																} `json:"all,omitempty"`
															} `json:"features,omitempty"`
															Sizes struct {
																Large struct {
																	H      int    `json:"h"`
																	W      int    `json:"w"`
																	Resize string `json:"resize"`
																} `json:"large"`
																Medium struct {
																	H      int    `json:"h"`
																	W      int    `json:"w"`
																	Resize string `json:"resize"`
																} `json:"medium"`
																Small struct {
																	H      int    `json:"h"`
																	W      int    `json:"w"`
																	Resize string `json:"resize"`
																} `json:"small"`
																Thumb struct {
																	H      int    `json:"h"`
																	W      int    `json:"w"`
																	Resize string `json:"resize"`
																} `json:"thumb"`
															} `json:"sizes"`
															OriginalInfo struct {
																Height     int `json:"height"`
																Width      int `json:"width"`
																FocusRects []struct {
																	X int `json:"x"`
																	Y int `json:"y"`
																	W int `json:"w"`
																	H int `json:"h"`
																} `json:"focus_rects"`
															} `json:"original_info"`
															MediaResults struct {
																Result struct {
																	MediaKey string `json:"media_key"`
																} `json:"result"`
															} `json:"media_results"`
															AdditionalMediaInfo struct {
																Monetizable bool `json:"monetizable"`
															} `json:"additional_media_info,omitempty"`
															AllowDownloadStatus struct {
																AllowDownload bool `json:"allow_download"`
															} `json:"allow_download_status,omitempty"`
															VideoInfo struct {
																AspectRatio    []int `json:"aspect_ratio"`
																DurationMillis int   `json:"duration_millis"`
																Variants       []struct {
																	ContentType string `json:"content_type"`
																	Url         string `json:"url"`
																	Bitrate     int    `json:"bitrate,omitempty"`
																} `json:"variants"`
															} `json:"video_info,omitempty"`
														} `json:"media"`
													} `json:"extended_entities,omitempty"`
													PossiblySensitive         bool   `json:"possibly_sensitive,omitempty"`
													PossiblySensitiveEditable bool   `json:"possibly_sensitive_editable,omitempty"`
													QuotedStatusIdStr         string `json:"quoted_status_id_str,omitempty"`
													QuotedStatusPermalink     struct {
														Url      string `json:"url"`
														Expanded string `json:"expanded"`
														Display  string `json:"display"`
													} `json:"quoted_status_permalink,omitempty"`
												} `json:"legacy"`
												QuotedStatusResult struct {
													Result struct {
														Typename string `json:"__typename"`
														RestId   string `json:"rest_id"`
														Core     struct {
															UserResults struct {
																Result struct {
																	Typename                   string `json:"__typename"`
																	Id                         string `json:"id"`
																	RestId                     string `json:"rest_id"`
																	AffiliatesHighlightedLabel struct {
																	} `json:"affiliates_highlighted_label"`
																	Avatar struct {
																		ImageUrl string `json:"image_url"`
																	} `json:"avatar"`
																	Core struct {
																		CreatedAt  string `json:"created_at"`
																		Name       string `json:"name"`
																		ScreenName string `json:"screen_name"`
																	} `json:"core"`
																	DmPermissions struct {
																		CanDm bool `json:"can_dm"`
																	} `json:"dm_permissions"`
																	HasGraduatedAccess bool `json:"has_graduated_access"`
																	IsBlueVerified     bool `json:"is_blue_verified"`
																	Legacy             struct {
																		DefaultProfile      bool   `json:"default_profile"`
																		DefaultProfileImage bool   `json:"default_profile_image"`
																		Description         string `json:"description"`
																		Entities            struct {
																			Description struct {
																				Urls []struct {
																					DisplayUrl  string `json:"display_url"`
																					ExpandedUrl string `json:"expanded_url"`
																					Url         string `json:"url"`
																					Indices     []int  `json:"indices"`
																				} `json:"urls"`
																			} `json:"description"`
																			Url struct {
																				Urls []struct {
																					DisplayUrl  string `json:"display_url"`
																					ExpandedUrl string `json:"expanded_url"`
																					Url         string `json:"url"`
																					Indices     []int  `json:"indices"`
																				} `json:"urls"`
																			} `json:"url,omitempty"`
																		} `json:"entities"`
																		FastFollowersCount      int           `json:"fast_followers_count"`
																		FavouritesCount         int           `json:"favourites_count"`
																		FollowersCount          int           `json:"followers_count"`
																		FriendsCount            int           `json:"friends_count"`
																		HasCustomTimelines      bool          `json:"has_custom_timelines"`
																		IsTranslator            bool          `json:"is_translator"`
																		ListedCount             int           `json:"listed_count"`
																		MediaCount              int           `json:"media_count"`
																		NormalFollowersCount    int           `json:"normal_followers_count"`
																		PinnedTweetIdsStr       []string      `json:"pinned_tweet_ids_str"`
																		PossiblySensitive       bool          `json:"possibly_sensitive"`
																		ProfileBannerUrl        string        `json:"profile_banner_url"`
																		ProfileInterstitialType string        `json:"profile_interstitial_type"`
																		StatusesCount           int           `json:"statuses_count"`
																		TranslatorType          string        `json:"translator_type"`
																		WantRetweets            bool          `json:"want_retweets"`
																		WithheldInCountries     []interface{} `json:"withheld_in_countries"`
																		Url                     string        `json:"url,omitempty"`
																	} `json:"legacy"`
																	Location struct {
																		Location string `json:"location"`
																	} `json:"location"`
																	MediaPermissions struct {
																		CanMediaTag bool `json:"can_media_tag"`
																	} `json:"media_permissions"`
																	ParodyCommentaryFanLabel string `json:"parody_commentary_fan_label"`
																	ProfileImageShape        string `json:"profile_image_shape"`
																	Privacy                  struct {
																		Protected bool `json:"protected"`
																	} `json:"privacy"`
																	RelationshipPerspectives struct {
																		Following bool `json:"following"`
																	} `json:"relationship_perspectives"`
																	TipjarSettings struct {
																		IsEnabled bool `json:"is_enabled,omitempty"`
																	} `json:"tipjar_settings"`
																	Verification struct {
																		Verified bool `json:"verified"`
																	} `json:"verification"`
																	Professional struct {
																		RestId           string `json:"rest_id"`
																		ProfessionalType string `json:"professional_type"`
																		Category         []struct {
																			Id       int    `json:"id"`
																			Name     string `json:"name"`
																			IconName string `json:"icon_name"`
																		} `json:"category"`
																	} `json:"professional,omitempty"`
																	SuperFollowEligible bool `json:"super_follow_eligible,omitempty"`
																} `json:"result"`
															} `json:"user_results"`
														} `json:"core"`
														UnmentionData struct {
														} `json:"unmention_data"`
														EditControl struct {
															EditTweetIds       []string `json:"edit_tweet_ids"`
															EditableUntilMsecs string   `json:"editable_until_msecs"`
															IsEditEligible     bool     `json:"is_edit_eligible"`
															EditsRemaining     string   `json:"edits_remaining"`
														} `json:"edit_control"`
														IsTranslatable bool `json:"is_translatable"`
														Views          struct {
															Count string `json:"count"`
															State string `json:"state"`
														} `json:"views"`
														Source    string `json:"source"`
														NoteTweet struct {
															IsExpandable     bool `json:"is_expandable"`
															NoteTweetResults struct {
																Result struct {
																	Id        string `json:"id"`
																	Text      string `json:"text"`
																	EntitySet struct {
																		Hashtags []interface{} `json:"hashtags"`
																		Symbols  []struct {
																			Indices []int  `json:"indices"`
																			Text    string `json:"text"`
																		} `json:"symbols"`
																		Urls         []interface{} `json:"urls"`
																		UserMentions []interface{} `json:"user_mentions"`
																	} `json:"entity_set"`
																	Richtext struct {
																		RichtextTags []struct {
																			FromIndex     int      `json:"from_index"`
																			ToIndex       int      `json:"to_index"`
																			RichtextTypes []string `json:"richtext_types"`
																		} `json:"richtext_tags"`
																	} `json:"richtext"`
																	Media struct {
																		InlineMedia []interface{} `json:"inline_media"`
																	} `json:"media"`
																} `json:"result"`
															} `json:"note_tweet_results"`
														} `json:"note_tweet,omitempty"`
														GrokAnalysisButton bool `json:"grok_analysis_button"`
														CommunityResults   struct {
															Result struct {
																Typename           string `json:"__typename"`
																IdStr              string `json:"id_str"`
																ViewerRelationship struct {
																	ModerationState struct {
																		Typename string `json:"__typename"`
																	} `json:"moderation_state"`
																} `json:"viewer_relationship"`
															} `json:"result"`
														} `json:"community_results,omitempty"`
														QuotedRefResult struct {
															Result struct {
																Typename string `json:"__typename"`
																RestId   string `json:"rest_id"`
															} `json:"result"`
														} `json:"quotedRefResult,omitempty"`
														Legacy struct {
															BookmarkCount     int    `json:"bookmark_count"`
															Bookmarked        bool   `json:"bookmarked"`
															CreatedAt         string `json:"created_at"`
															ConversationIdStr string `json:"conversation_id_str"`
															DisplayTextRange  []int  `json:"display_text_range"`
															Entities          struct {
																Hashtags []interface{} `json:"hashtags"`
																Media    []struct {
																	DisplayUrl           string `json:"display_url"`
																	ExpandedUrl          string `json:"expanded_url"`
																	IdStr                string `json:"id_str"`
																	Indices              []int  `json:"indices"`
																	MediaKey             string `json:"media_key"`
																	MediaUrlHttps        string `json:"media_url_https"`
																	Type                 string `json:"type"`
																	Url                  string `json:"url"`
																	ExtMediaAvailability struct {
																		Status string `json:"status"`
																	} `json:"ext_media_availability"`
																	Features struct {
																		Large struct {
																			Faces []interface{} `json:"faces"`
																		} `json:"large"`
																		Medium struct {
																			Faces []interface{} `json:"faces"`
																		} `json:"medium"`
																		Small struct {
																			Faces []interface{} `json:"faces"`
																		} `json:"small"`
																		Orig struct {
																			Faces []interface{} `json:"faces"`
																		} `json:"orig"`
																	} `json:"features"`
																	Sizes struct {
																		Large struct {
																			H      int    `json:"h"`
																			W      int    `json:"w"`
																			Resize string `json:"resize"`
																		} `json:"large"`
																		Medium struct {
																			H      int    `json:"h"`
																			W      int    `json:"w"`
																			Resize string `json:"resize"`
																		} `json:"medium"`
																		Small struct {
																			H      int    `json:"h"`
																			W      int    `json:"w"`
																			Resize string `json:"resize"`
																		} `json:"small"`
																		Thumb struct {
																			H      int    `json:"h"`
																			W      int    `json:"w"`
																			Resize string `json:"resize"`
																		} `json:"thumb"`
																	} `json:"sizes"`
																	OriginalInfo struct {
																		Height     int `json:"height"`
																		Width      int `json:"width"`
																		FocusRects []struct {
																			X int `json:"x"`
																			Y int `json:"y"`
																			W int `json:"w"`
																			H int `json:"h"`
																		} `json:"focus_rects"`
																	} `json:"original_info"`
																	AllowDownloadStatus struct {
																		AllowDownload bool `json:"allow_download"`
																	} `json:"allow_download_status"`
																	MediaResults struct {
																		Result struct {
																			MediaKey string `json:"media_key"`
																		} `json:"result"`
																	} `json:"media_results"`
																} `json:"media,omitempty"`
																Symbols []struct {
																	Indices []int  `json:"indices"`
																	Text    string `json:"text"`
																} `json:"symbols"`
																Timestamps   []interface{} `json:"timestamps"`
																Urls         []interface{} `json:"urls"`
																UserMentions []interface{} `json:"user_mentions"`
															} `json:"entities"`
															ExtendedEntities struct {
																Media []struct {
																	DisplayUrl           string `json:"display_url"`
																	ExpandedUrl          string `json:"expanded_url"`
																	IdStr                string `json:"id_str"`
																	Indices              []int  `json:"indices"`
																	MediaKey             string `json:"media_key"`
																	MediaUrlHttps        string `json:"media_url_https"`
																	Type                 string `json:"type"`
																	Url                  string `json:"url"`
																	ExtMediaAvailability struct {
																		Status string `json:"status"`
																	} `json:"ext_media_availability"`
																	Features struct {
																		Large struct {
																			Faces []interface{} `json:"faces"`
																		} `json:"large"`
																		Medium struct {
																			Faces []interface{} `json:"faces"`
																		} `json:"medium"`
																		Small struct {
																			Faces []interface{} `json:"faces"`
																		} `json:"small"`
																		Orig struct {
																			Faces []interface{} `json:"faces"`
																		} `json:"orig"`
																	} `json:"features"`
																	Sizes struct {
																		Large struct {
																			H      int    `json:"h"`
																			W      int    `json:"w"`
																			Resize string `json:"resize"`
																		} `json:"large"`
																		Medium struct {
																			H      int    `json:"h"`
																			W      int    `json:"w"`
																			Resize string `json:"resize"`
																		} `json:"medium"`
																		Small struct {
																			H      int    `json:"h"`
																			W      int    `json:"w"`
																			Resize string `json:"resize"`
																		} `json:"small"`
																		Thumb struct {
																			H      int    `json:"h"`
																			W      int    `json:"w"`
																			Resize string `json:"resize"`
																		} `json:"thumb"`
																	} `json:"sizes"`
																	OriginalInfo struct {
																		Height     int `json:"height"`
																		Width      int `json:"width"`
																		FocusRects []struct {
																			X int `json:"x"`
																			Y int `json:"y"`
																			W int `json:"w"`
																			H int `json:"h"`
																		} `json:"focus_rects"`
																	} `json:"original_info"`
																	AllowDownloadStatus struct {
																		AllowDownload bool `json:"allow_download"`
																	} `json:"allow_download_status"`
																	MediaResults struct {
																		Result struct {
																			MediaKey string `json:"media_key"`
																		} `json:"result"`
																	} `json:"media_results"`
																} `json:"media"`
															} `json:"extended_entities,omitempty"`
															FavoriteCount             int    `json:"favorite_count"`
															Favorited                 bool   `json:"favorited"`
															FullText                  string `json:"full_text"`
															IsQuoteStatus             bool   `json:"is_quote_status"`
															Lang                      string `json:"lang"`
															PossiblySensitive         bool   `json:"possibly_sensitive,omitempty"`
															PossiblySensitiveEditable bool   `json:"possibly_sensitive_editable,omitempty"`
															QuoteCount                int    `json:"quote_count"`
															QuotedStatusIdStr         string `json:"quoted_status_id_str,omitempty"`
															QuotedStatusPermalink     struct {
																Url      string `json:"url"`
																Expanded string `json:"expanded"`
																Display  string `json:"display"`
															} `json:"quoted_status_permalink,omitempty"`
															ReplyCount   int    `json:"reply_count"`
															RetweetCount int    `json:"retweet_count"`
															Retweeted    bool   `json:"retweeted"`
															UserIdStr    string `json:"user_id_str"`
															IdStr        string `json:"id_str"`
														} `json:"legacy"`
													} `json:"result"`
												} `json:"quoted_status_result,omitempty"`
											} `json:"result"`
										} `json:"tweet_results"`
										TweetDisplayType string `json:"tweetDisplayType"`
									} `json:"itemContent,omitempty"`
									ClientEventInfo struct {
										Component string `json:"component"`
										Element   string `json:"element"`
										Details   struct {
											TimelinesDetails struct {
												InjectionType string `json:"injectionType"`
											} `json:"timelinesDetails"`
										} `json:"details"`
									} `json:"clientEventInfo,omitempty"`
									Value      string `json:"value,omitempty"`
									CursorType string `json:"cursorType,omitempty"`
								} `json:"content"`
							} `json:"entries,omitempty"`
						} `json:"instructions"`
						Metadata struct {
							ScribeConfig struct {
								Page string `json:"page"`
							} `json:"scribeConfig"`
						} `json:"metadata"`
					} `json:"timeline"`
				} `json:"ranked_community_timeline"`
			} `json:"result"`
		} `json:"communityResults"`
	} `json:"data"`
}
