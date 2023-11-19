SELECT * FROM Auctions
WHERE item_id = 13452
AND faction_id = 0
AND connected_realm_id = 5284
AND timestamp >= strftime('%s', 'now', '-7 days')
ORDER BY buyout ASC;