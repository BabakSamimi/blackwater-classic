package blackwater

import (
	"database/sql"
	"log"
)

func InsertAuctions(db *sql.DB, auctionJson AuctionJson, importTime int64, connectedRealmID int, faction_id int) error {

	// Do something with the auction data
	tx, err := db.Begin()

	if err != nil {
		return err
	}

	stmt, err := tx.Prepare(`INSERT OR REPLACE INTO Auctions(
		auction_id, buyout, quantity, time_left,
		timestamp, 
		item_id, connected_realm_id, faction_id) VALUES(
		?, ?, ?, ?,
		?,
		?, ?, ?
		)`)

	if err != nil {
		return err
	}

	defer stmt.Close()

	counter := 0
	commitSize := 10000

	// Every batch should have the same import time
	// this might make it easier for SQL to sort
	for _, auction := range auctionJson.Auctions {

		_, err = stmt.Exec(auction.ID, auction.Buyout, auction.Quantity, auction.TimeLeft,
			importTime,
			auction.Item.ID, connectedRealmID, faction_id)

		if err != nil {
			tx.Rollback()
			return nil
		}

		counter++

		if counter >= commitSize {

			err = tx.Commit()
			if err != nil {
				log.Println("Could not commit")
				return err
			}
			tx, err = db.Begin()
			if err != nil {
				log.Println("Could not begin")
				return err
			}

			stmt, err = tx.Prepare(`INSERT OR REPLACE INTO Auctions(
				auction_id, buyout, quantity, time_left,
				timestamp, 
				item_id, connected_realm_id, faction_id) VALUES(
				?, ?, ?, ?,
				?,
				?, ?, ?
				)`)

			if err != nil {
				return err
			}

			counter = 0
		}

	}

	err = tx.Commit()

	if err != nil {
		return nil
	}

	log.Println("Finished importing auctions to DB.")

	return nil

}
