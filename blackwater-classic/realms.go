package blackwater

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/valyala/fasthttp"
)

func UpdateRealmTable(api *API, databaseFile string, server_json ServersJson) {

	db, err := OpenDB(databaseFile)

	if err != nil {
		log.Printf("Could not open DB: %q\n", err)
	}

	// Search for servers via the web API
	// Collect their metadata and store it in the DB
	for _, server := range server_json.Servers {

		log.Printf("Server name: %s\n", server.Name)

		time.Sleep(250 * time.Millisecond)

		// Search for the realm using the search feature of connected realms
		res, err := api.ConnectedRealmSearch(fmt.Sprintf("status.type=UP&realms.name.en_GB=%s", server.Name))

		if err != nil {
			log.Printf("Error searching for realm %s: %q\n", server.Name, err)
			continue
		}

		defer fasthttp.ReleaseResponse(res)

		var searchJson ConnectedRealmSearchJson
		err = json.Unmarshal(res.Body(), &searchJson)

		if err != nil {
			log.Printf("Error unmarshaling response from connected-realm search: %q\n", err)
		}

		if len(searchJson.Results) > 0 {
			ID := searchJson.Results[0].Data.ID
			log.Println("ID:", ID)

			// Fetch data about the connected realm
			// We will extract a href from this response that contains
			// hrefs to each auction house
			realmRes, err := api.ConnectedRealm(ID)

			if err != nil {
				log.Printf("Error fetching connected realm ID %d: %q\n", ID, err)
				continue
			}

			defer fasthttp.ReleaseResponse(realmRes)

			var realmJson ConnectedRealmJsonLite
			err = json.Unmarshal(realmRes.Body(), &realmJson)

			if err != nil {
				log.Printf("Error unmarshaling response: %q\n", err)
			}

			href := realmJson.Auctions.Href

			if len(href) > 0 {

				// Fetch hrefs for all the three auction houses for this realm
				// These hrefs will be stored in the SQL DB and be the link
				// that we use to fetch auction house data
				metadataRes, err := api.FetchFromHref(href)

				if err != nil {
					log.Printf("Error fetching metadata from %d: %q\n", ID, err)
					continue
				}

				defer fasthttp.ReleaseResponse(metadataRes)

				var ahMetadataJson AuctionHouseMetaDataJson
				err = json.Unmarshal(metadataRes.Body(), &ahMetadataJson)

				if err != nil {
					log.Printf("Error unmarshaling response: %q\n", err)
				}

				var alliance_href, horde_href, neutral_href string

				for _, meta := range ahMetadataJson.Auctions {
					log.Printf("meta.Key.Href = %s\n", meta.Key.Href)
					if meta.Name.En_GB == "Alliance Auction House" {
						alliance_href = meta.Key.Href
					} else if meta.Name.En_GB == "Horde Auction House" {
						horde_href = meta.Key.Href
					} else if meta.Name.En_GB == "Blackwater Auction House" {
						neutral_href = meta.Key.Href
					}
				}

				_, err = db.Exec(fmt.Sprintf("INSERT OR REPLACE INTO connectedRealms (connected_realm_id, region, name, alliance_ah_href, horde_ah_href, neutral_ah_href) VALUES ('%d', '%d', '%s', '%s', '%s', '%s')",
					ID, api.region, server.Name, alliance_href, horde_href, neutral_href))

				if err != nil {
					log.Printf("Could not insert/replace into the connectedRealms table: %q\n", err)
				}
			}
		}
	}

	db.Close()
}
