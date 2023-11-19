package blackwater

type DatabaseJson struct {
	DatabaseType     string `json:"database_type"`
	ConnectionString string `json:"connection_string"`
}

type ConnectedRealmsIndexJson struct {
	ConnectedRealms []struct {
		Href string `json:"href"`
	} `json:"connected_realms"`
}

type ConnectedRealmJson struct {
	ID     int `json:"id"`
	Realms []struct {
		ID           int    `json:"id"`
		Name         string `json:"name"`
		Timezone     string `json:"timezone"`
		IsTournament bool   `json:"is_tournament"`
	} `json:"realms"`
}

type ConnectedRealmJsonLite struct {
	Auctions struct {
		Href string `json:"href"`
	} `json:"auctions"`
}

type ConnectedRealmSearchJson struct {
	Results []struct {
		Data struct {
			ID int `json:"id"`
		} `json:"data"`
	} `json:"results"`
}

type AuctionHouseMetaDataJson struct {
	Auctions []struct {
		Key struct {
			Href string `json:"href"`
		} `json:"key"`

		Name struct {
			En_GB string `json:"en_GB"`
		} `json:"name"`

		ID int `json:"id"`
	} `json:"auctions"`
}

// An auction that you can find in the auction house
type AuctionJson struct {
	Auctions []struct {
		ID   int `json:"id"`
		Item struct {
			ID int `json:"id"`
		} `json:"item"`
		Buyout   int    `json:"buyout"`
		Quantity int    `json:"quantity"`
		TimeLeft string `json:"time_left"`
	} `json:"auctions"`
}

type ItemJson struct {
	Name          string `json:"name"`
	Level         int    `json:"level"`
	RequiredLevel int    `json:"required_level"`

	Quality struct {
		Name string `json:"name"`
	} `json:"quality"`

	ItemClass struct {
		Name string `json:"name"`
		ID   int    `json:"id"`
	} `json:"item_class"`

	ItemSubClass struct {
		Name string `json:"name"`
		ID   int    `json:"id"`
	} `json:"item_subclass"`

	SellPrice int `json:"sell_price"`
}
