package blackwater

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"time"

	"github.com/valyala/fasthttp"
)

const itemPrepareStatement = `INSERT OR REPLACE INTO Items(
	item_id, 
	item_class_id, item_class, 
	item_subclass_id, item_subclass,
	quality, name) VALUES(?, ?, ?, ?, ?, ?, ?)`

func CacheItems(api *API, db *sql.DB) error {

	rowsQuery, err := db.Query(`SELECT DISTINCT A.item_id
	FROM Auctions A
	LEFT JOIN Items I
	ON A.item_id = I.item_id
	WHERE I.item_id IS NULL
	ORDER BY A.item_id;`)

	if err != nil {
		log.Printf("Cannot cache items: %q\n", err)
	}

	itemIDs := []int{}

	defer rowsQuery.Close()

	for rowsQuery.Next() {
		var itemID int

		err = rowsQuery.Scan(&itemID)

		if err != nil {
			log.Fatal(err)
		}

		itemIDs = append(itemIDs, itemID)
	}

	err = rowsQuery.Err()

	if err != nil {
		log.Fatal(err)
	}

	tx, err := db.Begin()

	if err != nil {
		return err
	}

	stmt, err := tx.Prepare(itemPrepareStatement)

	if err != nil {
		return err
	}

	defer stmt.Close()

	counter := 0
	commitSize := 50
	failedCounter := 0

	for _, itemID := range itemIDs {
		if failedCounter > 5 {
			return errors.New("can't call the blizzard api at the moment")
		}
		res, err := api.ClassicItem(itemID)

		if err != nil {
			log.Printf("CacheItems tried to call ClassicItem(%d) but did not receive an accepted HTTP answer.\n", itemID)
			log.Println("Re-trying again after 10 seconds...")
			time.Sleep(10 * time.Second)

			log.Printf("Attempting to call ClassicItem(%d) again\n", itemID)
			res, err = api.ClassicItem(itemID)

			if err != nil {
				log.Println("CacheItems failed again...")
				failedCounter++
				continue
			}
		}

		defer fasthttp.ReleaseResponse(res)

		var itemJson ItemJson
		err = json.Unmarshal(res.Body(), &itemJson)

		if err != nil {
			return err
		}

		_, err = stmt.Exec(
			itemID,
			itemJson.ItemClass.ID, itemJson.ItemClass.Name,
			itemJson.ItemSubClass.ID, itemJson.ItemSubClass.Name,
			itemJson.Quality.Name,
			itemJson.Name)

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

			stmt, err = tx.Prepare(itemPrepareStatement)

			if err != nil {
				return err
			}

			counter = 0

		}

		time.Sleep(500 * time.Millisecond)

	}
	err = tx.Commit()

	if err != nil {
		return nil
	}

	log.Println("Finished caching items.")

	return nil
}
