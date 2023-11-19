package blackwater

import (
	"database/sql"
	"log"
)

type Database struct {
	DatabaseType     string
	ConnectionString string
	Handle           *sql.DB
}

type AuctionColumns struct {
	ConnectedRealmID int
	Region           int
	Name             string
	AllianceHref     string
	HordeHref        string
	NeutralHref      string
}

type ItemColumns struct {
	ItemID         int
	itemClassID    int
	ItemClass      string
	ItemSubclassID int
	ItemSubclass   string
	Quality        string
	Name           string
}

func SetupDatabase(db *Database) error {

	handle, err := sql.Open(db.DatabaseType, db.ConnectionString)
	if err != nil {
		return err
	}

	defer handle.Close()

	log.Print("Created a handle to the DB.")

	/*
		region := { EU = 0, US = 1}
	*/
	_, err = handle.Exec(`CREATE TABLE IF NOT EXISTS ConnectedRealms(
		connected_realm_id INTEGER NOT NULL PRIMARY KEY,
		region INTEGER,
		name TEXT,
		timezone TEXT,
		alliance_ah_href TEXT,
		horde_ah_href TEXT,
		neutral_ah_href TEXT);`)

	if err != nil {
		return err
	}

	/*

		Item Class ID: Weapon, head, shoulder, etc
		Item Subclass ID: Stave, Sword etc.
		Example:
		Item Class ID: Consumable
		Item Subclass ID: Elixir, potion, enchant, flask, ....

	*/
	_, err = handle.Exec(`CREATE TABLE IF NOT EXISTS Items(
		item_id INTEGER NOT NULL PRIMARY KEY,
		item_class_id INTEGER,
		item_class STRING,
		item_subclass_id INTEGER,
		item_subclass STRING,
		quality STRING,
		name TEXT);`)

	if err != nil {
		return err
	}

	log.Println("Created Items table")

	_, err = handle.Exec(`CREATE TABLE IF NOT EXISTS Factions(
		faction_id INTEGER PRIMARY KEY,
		faction_name TEXT);`)

	if err != nil {
		return err
	}

	_, err = handle.Exec(`INSERT INTO Factions(faction_id, faction_name) VALUES(0, "Alliance")`)
	if err != nil {
		log.Println(err)
	}

	_, err = handle.Exec(`INSERT INTO Factions(faction_id, faction_name) VALUES(1, "Horde")`)
	if err != nil {
		log.Println(err)
	}

	_, err = handle.Exec(`INSERT INTO Factions(faction_id, faction_name) VALUES(2, "Neutral")`)
	if err != nil {
		log.Println(err)
	}

	log.Println("Created factions table")

	_, err = handle.Exec(`CREATE TABLE IF NOT EXISTS Auctions(
		id INTEGER NOT NULL PRIMARY KEY,
		auction_id INTEGER,
		buyout INTEGER,
		quantity INTEGER,
		time_left TEXT,
		timestamp DATETIME,
		item_id INTEGER,
		connected_realm_id INTEGER,
		faction_id INTEGER,
		FOREIGN KEY(connected_realm_id) REFERENCES ConnectedRealms(connected_realm_id),
		FOREIGN KEY(item_id) REFERENCES Items(item_id),
		FOREIGN KEY(faction_id) REFERENCES Factions(faction_id),
		UNIQUE(auction_id, connected_realm_id, time_left));`)

	if err != nil {
		return err
	}

	log.Println("Created Auctions table")

	/*
			_, err = handle.Exec(`CREATE TABLE IF NOT EXISTS Stats(
				id INTEGER NOT NULL PRIMARY KEY,
				item_id INTEGER,
				connected_realm_id INTEGER,
				mean_price INTEGER,
				median_price INTEGER,
				min_price INTEGER,
				FOREIGN KEY(connected_realm_id) REFERENCES ConnectedRealms(connected_realm_id),
				FOREIGN KEY(item_id) REFERENCES Items(item_id))`)

			if err != nil {
				return err
			}


		_, err = handle.Exec(`CREATE TABLE IF NOT EXISTS WeeklySeries(
				id INTEGER NOT NULL PRIMARY KEY,
				item_id INTEGER,
				connected_realm_id INTEGER,
				mean_price INTEGER,
				median_price INTEGER,
				min_price INTEGER,
				FOREIGN KEY(connected_realm_id) REFERENCES ConnectedRealms(connected_realm_id),
				FOREIGN KEY(item_id) REFERENCES Items(item_id))`)

		if err != nil {
			return err
		}

	*/

	return nil
}

func NewLocalDatabase(p string) (db Database) {

	db.DatabaseType = "sqlite3"
	db.ConnectionString = p

	return
}

func (db *Database) OpenConnection() (err error) {
	err = nil

	db.Handle, err = sql.Open(db.DatabaseType, db.ConnectionString)

	return
}

func (db *Database) CloseConnection() {
	db.Handle.Close()
}

func OpenDB(dbPath string) (*sql.DB, error) {
	log.Println("Opening a handle to SQL DB.")

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	log.Print("Created a handle to the DB.")

	return db, nil
}
