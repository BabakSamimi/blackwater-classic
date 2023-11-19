-- SQLite
SELECT DISTINCT A.item_id 
FROM Auctions A 
LEFT JOIN Items I 
ON A.item_id = I.item_id 
WHERE I.item_id IS NULL 
ORDER BY A.item_id;