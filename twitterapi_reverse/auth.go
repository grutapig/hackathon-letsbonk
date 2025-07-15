package twitterapi_reverse

import (
	"regexp"
	"strings"
)

type TwitterAuth struct {
	Authorization string `json:"authorization"`
	XCSRFToken    string `json:"x_csrf_token"`
	Cookie        string `json:"cookie"`
}

func ParseFromCurl(curlCommand string) (*TwitterAuth, error) {
	auth := &TwitterAuth{}

	authRegex := regexp.MustCompile(`-H\s+['"]authorization:\s*Bearer\s+([^'"]+)['"]`)
	csrfRegex := regexp.MustCompile(`-H\s+['"]x-csrf-token:\s*([^'"]+)['"]`)
	cookieRegex := regexp.MustCompile(`-H\s+['"]Cookie:\s*([^'"]+)['"]`)

	if matches := authRegex.FindStringSubmatch(curlCommand); len(matches) > 1 {
		auth.Authorization = "Bearer " + matches[1]
	}

	if matches := csrfRegex.FindStringSubmatch(curlCommand); len(matches) > 1 {
		auth.XCSRFToken = matches[1]
	}

	if matches := cookieRegex.FindStringSubmatch(curlCommand); len(matches) > 1 {
		auth.Cookie = matches[1]
	}

	return auth, nil
}

func ParseFromHeaders(headers map[string]string) (*TwitterAuth, error) {
	auth := &TwitterAuth{}

	for key, value := range headers {
		lowerKey := strings.ToLower(key)
		switch lowerKey {
		case "authorization":
			auth.Authorization = value
		case "x-csrf-token":
			auth.XCSRFToken = value
		case "cookie":
			auth.Cookie = value
		}
	}

	return auth, nil
}

func NewTwitterAuth(authorization, xcsrfToken, cookie string) *TwitterAuth {
	return &TwitterAuth{
		Authorization: authorization,
		XCSRFToken:    xcsrfToken,
		Cookie:        cookie,
	}
}
