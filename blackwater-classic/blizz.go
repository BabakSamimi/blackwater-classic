package blackwater

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"time"

	"github.com/valyala/fasthttp"
)

type Token struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in,omitempty"`
}

type Client struct {
	ID     string
	Secret string

	Token *Token
}

type Region int
type Locale string
type Namespace string
type Endpoint string

const (
	Era = iota
	Wrath
	Retail
)

const (
	EU = iota
	US
)

const (
	EnGB Locale = "en_GB"
	EnUS Locale = "en_US"
)

var RegionStrings = [2]string{"eu", "us"}
var NamespaceStrings = [2][2]string{
	{"dynamic-classic1x-eu", "static-classic1x-eu"},
	{"dynamic-classic1x-us", "static-classic1x-us"},
}

type API struct {
	User        Client
	httpClient  *fasthttp.Client
	region      Region
	locale      Locale
	gameVersion int
}

// Creates a new client
func NewAPI(clientID string, clientSecret string) (api *API, err error) {

	if clientID == "" || clientSecret == "" {
		return nil, errors.New("Client ID or Client Secret was empty")
	}

	api = &API{}
	api.User.ID = clientID
	api.User.Secret = clientSecret

	api.httpClient = &fasthttp.Client{
		NoDefaultUserAgentHeader:      true,
		DisableHeaderNamesNormalizing: true,
		// increase DNS cache time to an hour instead of default minute
		Dial: (&fasthttp.TCPDialer{
			Concurrency:      4096,
			DNSCacheDuration: time.Hour,
		}).Dial,
	}

	// Default
	api.SetRegion(EU, EnGB)
	log.Printf("Current locale: %s\n", api.locale)

	// Check if token is cached or if it needs to be refreshed
	f, err := os.OpenFile("blackwater.oauth", os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0644)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	log.Println("Reading from oauth file to see if there is a cached token")
	fileBuffer, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}

	if len(fileBuffer) > 1 {

		err = json.Unmarshal(fileBuffer, &api.User.Token)
		if err == nil {
			// TODO: Verify that the JSON fields contains any data
			log.Println("Found a cached oauth token")

			tokenExpiration := time.Unix(int64(api.User.Token.ExpiresIn), 0)
			nowUnix := time.Now().Unix()

			// Add a margin of 10 seconds, in case we are unlucky
			// and we try to fetch with an expired token

			if nowUnix < (tokenExpiration.Unix() + 10) {
				log.Println("Token has not yet expired")
				log.Println("Token will expire:", tokenExpiration)
				return api, nil
			}
		}
	}

	log.Println("Found no token or it expired or something went wrong. Fetching a new token")

	req := fasthttp.AcquireRequest()
	url := fasthttp.AcquireURI()

	// Set URL
	url.Parse(nil, []byte("https://oauth.battle.net/token"))
	url.SetUsername(clientID)
	url.SetPassword(clientSecret)
	req.SetURI(url)
	fasthttp.ReleaseURI(url)

	req.Header.SetMethod(fasthttp.MethodPost)
	req.SetBody([]byte("grant_type=client_credentials"))

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	res := fasthttp.AcquireResponse()
	err = api.httpClient.Do(req, res)
	fasthttp.ReleaseRequest(req)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(res.Body(), &api.User.Token)
	fasthttp.ReleaseResponse(res)

	if err != nil {
		return nil, err
	}

	api.User.Token.ExpiresIn += int(time.Now().Unix())

	log.Println("Newly fetched oauth token will expire at:", time.Unix(int64(api.User.Token.ExpiresIn), 0))

	j, _ := json.Marshal(api.User.Token)
	_, err = f.Write(j)
	if err != nil {
		return nil, err
	}

	return
}

func (api *API) SetRegion(region Region, locale Locale) {
	api.region = region
	api.locale = locale
}

func (api *API) SetGameVersion(gv int) {
	api.gameVersion = gv
}

func (api *API) buildUrlDynamic(endpoint string) string {
	return fmt.Sprintf(
		"https://%s.api.blizzard.com/%s?namespace=%s&locale=%s&access_token=%s",
		RegionStrings[api.region],
		endpoint,
		NamespaceStrings[api.region][0],
		api.locale,
		api.User.Token.AccessToken)
}

func (api *API) buildUrlDynamicWithParameters(endpoint string, parameters string) string {
	return fmt.Sprintf(
		"https://%s.api.blizzard.com/%s?namespace=%s&locale=%s&%s&access_token=%s",
		RegionStrings[api.region],
		endpoint,
		NamespaceStrings[api.region][0],
		api.locale,
		parameters,
		api.User.Token.AccessToken)
}

func (api *API) buildUrlStatic(endpoint string) string {
	return fmt.Sprintf(
		"https://%s.api.blizzard.com/%s?namespace=%s&locale=%s&access_token=%s",
		RegionStrings[api.region],
		endpoint,
		NamespaceStrings[api.region][1],
		api.locale,
		api.User.Token.AccessToken)
}

// Following types are used for JSON unmarshaling

// This is not an archetype used for Blizzard Web API
// This is used to unmarshal from [eu|us]-servers.json files
type ServersJson struct {
	Servers []struct {
		Name   string   `json:"name"`
		Houses []string `json:"houses"`
	} `json:"servers"`
}

func (api *API) fetchData(u string) (*fasthttp.Response, error) {

	// Parse the URL
	{
		parsedURL, err := url.Parse(u)
		if err != nil {
			fmt.Println("Error parsing URL:", err)
			return nil, err
		}

		// Remove the access_token parameter
		q, _ := url.ParseQuery(parsedURL.RawQuery)
		q.Del("access_token")

		// Reconstruct the URL without the access_token parameter
		parsedURL.RawQuery = q.Encode()
		logSafeUrl := parsedURL.String()

		log.Printf("GET on %s\n", logSafeUrl)
	}

	req := fasthttp.AcquireRequest()
	url := fasthttp.AcquireURI()

	url.Parse(nil, []byte(u))
	req.SetURI(url)
	fasthttp.ReleaseURI(url)

	req.Header.SetMethod(fasthttp.MethodGet)
	req.Header.Set("Accept", "application/json")

	res := fasthttp.AcquireResponse()
	err := api.httpClient.Do(req, res)
	fasthttp.ReleaseRequest(req)

	if err != nil {
		fasthttp.ReleaseResponse(res)
		return nil, err
	}

	if res.Header.StatusCode() != fasthttp.StatusOK {
		fasthttp.ReleaseResponse(res)
		return nil, errors.New("did not return ok status")
	}

	return res, nil

}

func (api *API) fetchDataCompressed(u string) (*fasthttp.Response, error) {

	// Parse the URL
	{
		parsedURL, err := url.Parse(u)
		if err != nil {
			fmt.Println("Error parsing URL:", err)
			return nil, err
		}

		// Remove the access_token parameter
		q, _ := url.ParseQuery(parsedURL.RawQuery)
		q.Del("access_token")

		// Reconstruct the URL without the access_token parameter
		parsedURL.RawQuery = q.Encode()
		logSafeUrl := parsedURL.String()

		log.Printf("GET on %s\n", logSafeUrl)
	}

	req := fasthttp.AcquireRequest()
	url := fasthttp.AcquireURI()

	url.Parse(nil, []byte(u))
	req.SetURI(url)
	fasthttp.ReleaseURI(url)

	req.Header.SetMethod(fasthttp.MethodGet)
	req.Header.Set("Accept-Encoding", "gzip")

	res := fasthttp.AcquireResponse()
	err := api.httpClient.Do(req, res)
	fasthttp.ReleaseRequest(req)

	if err != nil {
		fasthttp.ReleaseResponse(res)
		return nil, err
	}

	if res.Header.StatusCode() != fasthttp.StatusOK {
		return nil, errors.New(res.String())
	}

	return res, nil

}

func (api *API) FetchFromHref(href string) (*fasthttp.Response, error) {
	if len(href) == 0 {
		return nil, errors.New("href is empty")
	}

	return api.fetchData(fmt.Sprintf(
		"%s&access_token=%s",
		href,
		api.User.Token.AccessToken))
}

func (api *API) FetchCompressedFromHref(href string) (*fasthttp.Response, error) {
	if len(href) == 0 {
		return nil, errors.New("href is empty")
	}

	return api.fetchDataCompressed(fmt.Sprintf(
		"%s&access_token=%s",
		href,
		api.User.Token.AccessToken))
}

func (api *API) ConnectedRealmsIndex() (*fasthttp.Response, error) {

	res, err := api.fetchData(
		api.buildUrlDynamic("data/wow/connected-realm/index"))

	return res, err

}

func (api *API) ConnectedRealm(connectedRealmID int) (*fasthttp.Response, error) {

	res, err := api.fetchData(
		api.buildUrlDynamic(fmt.Sprintf("data/wow/connected-realm/%d", connectedRealmID)))

	return res, err

}

func (api *API) ConnectedRealmSearch(parameters string) (*fasthttp.Response, error) {

	res, err := api.fetchData(
		api.buildUrlDynamicWithParameters("data/wow/search/connected-realm", parameters))

	return res, err

}

func (api *API) ClassicAuctions(connectedRealmID int, auctionHouseID int) (*fasthttp.Response, error) {

	res, err := api.fetchDataCompressed(
		api.buildUrlDynamic(fmt.Sprintf("data/wow/connected-realm/%d/auctions/%d", connectedRealmID, auctionHouseID)))

	return res, err

}

func (api *API) ClassicAuctionHouseIndex(realmID int) (*fasthttp.Response, error) {

	res, err := api.fetchDataCompressed(
		api.buildUrlDynamic(fmt.Sprintf("data/wow/connected-realm/%d/auctions/index", realmID)))

	return res, err

}

func (api *API) ClassicItem(itemID int) (*fasthttp.Response, error) {
	res, err := api.fetchData(api.buildUrlStatic(fmt.Sprintf("data/wow/item/%d", itemID)))

	return res, err
}
