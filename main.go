package main

import (
	"blackwater/blackwater-classic"
	"database/sql"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/mattn/go-sqlite3"
	"github.com/valyala/fasthttp"
)

func ReadEntireFile(p string) ([]byte, error) {

	log.Println("Reading from:", p)

	file, err := os.OpenFile(p, os.O_RDONLY, 0644)

	if err != nil {
		return nil, err
	}

	defer file.Close()

	bytes, err := ioutil.ReadAll(file)

	if err != nil {
		log.Println("Error when trying to read servers.json")
		return nil, err
	}

	return bytes, nil
}

func ReadDatabaseConfig(p string) (blackwater.Database, error) {

	var result blackwater.Database
	err := FileExists(p)

	if err != nil {
		return result, err
	}

	bytes, err := ReadEntireFile(p)

	if err != nil {
		return result, err
	}

	var databaseJson blackwater.DatabaseJson
	err = json.Unmarshal(bytes, &databaseJson)

	if err != nil {
		log.Println("Error when trying to decode the json data")
		return result, err
	}

	if len(databaseJson.DatabaseType) == 0 || len(databaseJson.ConnectionString) == 0 {
		return result, errors.New("one of the fields in the json database config file is empty")
	}

	result.DatabaseType = databaseJson.DatabaseType
	result.ConnectionString = databaseJson.ConnectionString

	return result, nil
}

func ReadServerConfig(api *blackwater.API, serverPath string) error {
	log.Println("Reading from:", serverPath)

	bytes, err := ReadEntireFile(serverPath)

	if err != nil {
		return err
	}

	var server_json blackwater.ServersJson
	err = json.Unmarshal(bytes, &server_json)

	if err != nil {
		log.Println("Error when trying to decode the json data")
		return err
	}

	blackwater.UpdateRealmTable(api, databaseFile, server_json)

	return nil
}

func FetchFactionAH(api *blackwater.API, db *sql.DB, href string, importTime int64, connectedRealmID int, faction_id int) (int, error) {
	var auctionJson blackwater.AuctionJson
	response, err := api.FetchCompressedFromHref(href)

	auctionCount := 0

	if err != nil {
		return auctionCount, err
	}

	data, err := response.BodyGunzip()

	if err != nil {
		return auctionCount, err
	}

	log.Println("Marshaling")
	err = json.Unmarshal(data, &auctionJson)
	log.Println("Marshaling done")
	auctionCount = len(auctionJson.Auctions)

	if err != nil {
		log.Printf("Could not unmarshal the AH data\n%q\n", err)
	} else {
		err = blackwater.InsertAuctions(db, auctionJson, importTime, connectedRealmID, faction_id)

		if err != nil {
			return 0, err
		}
	}

	fasthttp.ReleaseResponse(response)

	return auctionCount, nil
}

func FileExists(p string) error {
	if _, err := os.Stat(p); os.IsNotExist(err) {
		return err
	}

	return nil
}

const (
	databaseFolder = "data/db"
	databaseFile   = databaseFolder + "/blackwater.db" // This is where all the data will go
)

func main() {

	f, err := os.OpenFile("blackwater.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}

	log.SetOutput(f)
	defer f.Close()

	//update := flag.NewFlagSet("update", flag.ExitOnError)

	initCmd := flag.NewFlagSet("init", flag.ExitOnError)
	initSql := initCmd.Bool("sql", false, "Sets up the database if it doesn't exist, using sqllite3.")

	flag.Parse()

	if len(os.Args) < 2 {
		fmt.Println("Expected: blackwater [init|run|update] [flags]")
		os.Exit(1)
	}

	os.Mkdir("data", 0777)

	api, apiCreationError := blackwater.NewAPI(os.Getenv("CLIENT_ID"), os.Getenv("CLIENT_SECRET"))
	if apiCreationError != nil {
		log.Fatal(apiCreationError)
	}

	log.Println("Successfully created an API client.")
	log.Println("Setting Game Version to Classic Era")

	api.SetGameVersion(blackwater.Era)

	database, err := ReadDatabaseConfig("db.json")

	if err != nil {
		log.Printf("ReadDatabaseConfig failed: %q\n", err)
		log.Println("Using a local sqllite3 database instead.")
		database = blackwater.NewLocalDatabase(databaseFile)
	}

	if os.Args[1] == "init" {
		initCmd.Parse(os.Args[2:])
		log.Println("Creating database.")

		// TODO: SQL init
		if *initSql {
			err = blackwater.SetupDatabase(&database)

			if err != nil {
				log.Print(err)
			}

		}

	} else if os.Args[1] == "update" {

		api.SetRegion(blackwater.EU, blackwater.EnGB)
		ReadServerConfig(api, "eu-servers.json")

		api.SetRegion(blackwater.US, blackwater.EnUS)
		ReadServerConfig(api, "us-servers.json")

	} else if os.Args[1] == "auctions" {

		err := database.OpenConnection()

		if err != nil {
			log.Printf("Could not open DB.\n")
			log.Fatal(err)
		}

		// 1. Look up every row in the realm table
		rowsQuery, err := database.Handle.Query("SELECT connected_realm_id, region, name, alliance_ah_href, horde_ah_href, neutral_ah_href FROM ConnectedRealms")
		if err != nil {
			log.Fatal(err)
		}

		rows := []blackwater.AuctionColumns{}

		defer rowsQuery.Close()

		for rowsQuery.Next() {
			var columns blackwater.AuctionColumns

			err = rowsQuery.Scan(&columns.ConnectedRealmID,
				&columns.Region,
				&columns.Name,
				&columns.AllianceHref,
				&columns.HordeHref,
				&columns.NeutralHref)

			if err != nil {
				log.Fatal(err)
			}

			rows = append(rows, columns)
		}

		err = rowsQuery.Err()

		if err != nil {
			log.Fatal(err)
		}

		numberOfAuctionsImported := 0
		importTime := time.Now().Unix()

		for _, row := range rows {

			// Fetch AH data using their hrefs
			// FetchFactionAH() will also save the auctions in the database

			allianceAuctionsCount, err := FetchFactionAH(api, database.Handle, row.AllianceHref, importTime, row.ConnectedRealmID, 0)

			if err != nil {
				log.Println(err)
			} else {
				log.Printf("Imported %d alliance auctions to the DB for %s (%d)\n", allianceAuctionsCount, row.Name, row.ConnectedRealmID)

			}

			hordeAuctionsCount, err := FetchFactionAH(api, database.Handle, row.HordeHref, importTime, row.ConnectedRealmID, 1)

			if err != nil {
				log.Println(err)
			} else {
				log.Printf("Imported %d horde auctions to the DB for %s (%d)\n", hordeAuctionsCount, row.Name, row.ConnectedRealmID)

			}

			NeutralAuctionsCount, err := FetchFactionAH(api, database.Handle, row.NeutralHref, importTime, row.ConnectedRealmID, 2)

			if err != nil {
				log.Println(err)
			} else {
				log.Printf("Imported %d neutral auctions to the DB for %s (%d)\n", NeutralAuctionsCount, row.Name, row.ConnectedRealmID)

			}

			numberOfAuctionsImported += allianceAuctionsCount
			numberOfAuctionsImported += hordeAuctionsCount
			numberOfAuctionsImported += NeutralAuctionsCount

		}

		database.CloseConnection()

		log.Println("Finished downloading auction house data.")
		log.Printf("Imported a total of %d auctions\n", numberOfAuctionsImported)
	} else if os.Args[1] == "items" {
		// Scan through the Auctions table to see if there is an item in there that is not cached, i.e in the items table
		err := database.OpenConnection()

		if err != nil {
			log.Printf("Could not open DB.\n")
			log.Fatal(err)
		}

		err = blackwater.CacheItems(api, database.Handle)

		if err != nil {
			log.Fatal(err)
		}

	} else if os.Args[1] == "reset-realms" {

		err := database.OpenConnection()

		if err != nil {
			log.Printf("Could not open DB.\n")
			log.Fatal(err)
		}

		_, err = database.Handle.Exec(`DELETE FROM ConnectedRealms`)

		if err != nil {
			log.Fatal(err)
		}

		log.Println("Deleted all records of realms.")
	}

}
