package twitterapi_reverse

import (
	"encoding/json"
	"fmt"
	"github.com/grutapig/hackaton/twitterapi"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestParseFromCurl(t *testing.T) {
	curlString := `curl 'https://x.com/i/api/graphql/f_muosmN8WvS9muD_kmwxA/CommunityTweetsTimeline?variables=%7B%22communityId%22%3A%221921225336253042983%22%2C%22count%22%3A20%2C%22displayLocation%22%3A%22Community%22%2C%22rankingMode%22%3A%22Relevance%22%2C%22withCommunity%22%3Atrue%7D&features=%7B%22rweb_video_screen_enabled%22%3Afalse%2C%22payments_enabled%22%3Afalse%2C%22profile_label_improvements_pcf_label_in_post_enabled%22%3Atrue%2C%22rweb_tipjar_consumption_enabled%22%3Atrue%2C%22verified_phone_label_enabled%22%3Afalse%2C%22creator_subscriptions_tweet_preview_api_enabled%22%3Atrue%2C%22responsive_web_graphql_timeline_navigation_enabled%22%3Atrue%2C%22responsive_web_graphql_skip_user_profile_image_extensions_enabled%22%3Afalse%2C%22premium_content_api_read_enabled%22%3Afalse%2C%22communities_web_enable_tweet_community_results_fetch%22%3Atrue%2C%22c9s_tweet_anatomy_moderator_badge_enabled%22%3Atrue%2C%22responsive_web_grok_analyze_button_fetch_trends_enabled%22%3Afalse%2C%22responsive_web_grok_analyze_post_followups_enabled%22%3Atrue%2C%22responsive_web_jetfuel_frame%22%3Atrue%2C%22responsive_web_grok_share_attachment_enabled%22%3Atrue%2C%22articles_preview_enabled%22%3Atrue%2C%22responsive_web_edit_tweet_api_enabled%22%3Atrue%2C%22graphql_is_translatable_rweb_tweet_is_translatable_enabled%22%3Atrue%2C%22view_counts_everywhere_api_enabled%22%3Atrue%2C%22longform_notetweets_consumption_enabled%22%3Atrue%2C%22responsive_web_twitter_article_tweet_consumption_enabled%22%3Atrue%2C%22tweet_awards_web_tipping_enabled%22%3Afalse%2C%22responsive_web_grok_show_grok_translated_post%22%3Afalse%2C%22responsive_web_grok_analysis_button_from_backend%22%3Afalse%2C%22creator_subscriptions_quote_tweet_preview_enabled%22%3Afalse%2C%22freedom_of_speech_not_reach_fetch_enabled%22%3Atrue%2C%22standardized_nudges_misinfo%22%3Atrue%2C%22tweet_with_visibility_results_prefer_gql_limited_actions_policy_enabled%22%3Atrue%2C%22longform_notetweets_rich_text_read_enabled%22%3Atrue%2C%22longform_notetweets_inline_media_enabled%22%3Atrue%2C%22responsive_web_grok_image_annotation_enabled%22%3Atrue%2C%22responsive_web_grok_community_note_auto_translation_is_enabled%22%3Afalse%2C%22responsive_web_enhance_cards_enabled%22%3Afalse%7D' \
  -H 'accept: */*' \
  -H 'accept-language: ru,en-US;q=0.9,en;q=0.8' \
  -H 'authorization: Bearer AAAAAAAAAAAAAAAAAAAAANRILgAAAAAAnNwIzUejRCOuH5E6I8xnZz4puTs%3D1Zv7ttfk8LF81IUq16cHjhLTvJu4FA33AGWWjCpTnA' \
  -H 'content-type: application/json' \
  -H 'Cookie: guest_id=v1%3A174887694015957546; kdt=NF6FKaUf1mZJpT9bmfCekQNxw14iOjmdxIs61KH2; auth_token=5b4d61da646d91271127948b48b9f41aa5f048e2; ct0=e7615092e354d4db4e30937e9c511fcbb2273d9856214b99aa07b4e3720cd67fccfa9734f7b9bb144a47286a4b3c796b3a8f686de33d24d596e393c675be0bd268376b978cdab3f27a28850c6b3557dd; twid=u%3D1936113633320165376; d_prefs=MjoxLGNvbnNlbnRfdmVyc2lvbjoyLHRleHRfdmVyc2lvbjoxMDAw; lang=ru; twtr_pixel_opt_in=N; _twitter_sess=BAh7CCIKZmxhc2hJQzonQWN0aW9uQ29udHJvbGxlcjo6Rmxhc2g6OkZsYXNo%250ASGFzaHsABjoKQHVzZWR7ADofbGFzdF9wYXNzd29yZF9jb25maXJtYXRpb24i%250AFTE3NTI2NjI1NDEzMTAwMDA6HnBhc3N3b3JkX2NvbmZpcm1hdGlvbl91aWQi%250AGDE5MzYxMTM2MzMzMjAxNjUzNzY%253D--82cf65f86a871c00c1e92045321ec14ebdaf7a81; att=1-7MVOhYKKTNzWKccqZCW1j7OMqCHuUzIPZS0Edq35; __cf_bm=eGwZdrsqBCknuNGYv24TAxtD5uQ_cOIsFqJvR74YmTQ-1752671849-1.0.1.1-S2_lrZZTZtzeDFkvi43YAvlnVCnekF8ZQO0tdjS3Zzhaj_MTIUtU1jWwsccm42gsYvown_vd6UqqUvTQ36LGY.EI2TgXmeSThFKlIgofpjk' \
  -H 'priority: u=1, i' \
  -H 'referer: https://x.com/i/communities/1921225336253042983' \
  -H 'sec-ch-ua: "Not)A;Brand";v="8", "Chromium";v="138", "Google Chrome";v="138"' \
  -H 'sec-ch-ua-mobile: ?0' \
  -H 'sec-ch-ua-platform: "Windows"' \
  -H 'sec-fetch-dest: empty' \
  -H 'sec-fetch-mode: cors' \
  -H 'sec-fetch-site: same-origin' \
  -H 'user-agent: Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/138.0.0.0 Safari/537.36' \
  -H 'x-client-transaction-id: nqs/23q/IOxKb2HsUBeM7JXE1L94sR22Md4fwIsz5sKwnLb85Bmu3S4xEyjjttROWBvdtponqhqCBz4hF8XhdXEBsZ0XnQ' \
-H 'x-csrf-token: e7615092e354d4db4e30937e9c511fcbb2273d9856214b99aa07b4e3720cd67fccfa9734f7b9bb144a47286a4b3c796b3a8f686de33d24d596e393c675be0bd268376b978cdab3f27a28850c6b3557dd' \
  -H 'x-twitter-active-user: yes' \
  -H 'x-twitter-auth-type: OAuth2Session' \
  -H 'x-twitter-client-language: ru' \
  -H 'x-xp-forwarded-for: b308803381e1fba8509deeea77bd81f0d84db7654c288136239550ece334e7be3af34f55e2a3dd3cf2d9f921bc405ae5e5fd69c95bbe6460a6e72d6788d5e66c4392e62d18ce7aa60b0fc55d1d2a6aae28420cc4e663ee998cdbe80d28443d39016f014eae2a80d12650bb1a509835457e0544cbac59e2beca9f7bb3b8c546e3a78f94a58c01c01bb7a24b831210ba75506525aa429f8b746ee4614595dfec4fc99e2c170fee68780092485a549f2a2fda402901131573fbb2c843b22092866d23b65723cfb2ab35ca2f757f1f53ed1a53c7c1ea01f6ad0444663be2eb71b4e9aed17d1d59bc5eaf29acab31517a4079a6b2f2dd8c27eda3621223'`
	auth, err := ParseFromCurl(curlString)
	fmt.Println(auth, err)
	fmt.Println(auth.Authorization)
	fmt.Println(auth.XCSRFToken)
	fmt.Println(auth.Cookie)
	godotenv.Load("../.env")
	service := NewTwitterReverseService(auth, os.Getenv(twitterapi.ENV_PROXY_DSN), false)
	data, err := service.GetCommunityTweets("1914102634241577036", 20)
	indent, err := json.MarshalIndent(data, "", "\t")
	fmt.Println(string(indent))
	assert.NoError(t, err)
}
