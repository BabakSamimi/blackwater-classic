SELECT A.*
FROM Auctions A
JOIN ConnectedRealms R ON A.connected_realm_id = R.connected_realm_id
WHERE R.region = 1
ORDER BY A.item_id;