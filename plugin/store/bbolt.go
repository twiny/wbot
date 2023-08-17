package store

// var (
// 	prefix = "store"
// )

// // BStore
// type BStore struct {
// 	prefix string
// 	db     *bbolt.DB
// }

// // NewBStore
// func NewBStore(db *bbolt.DB) (wbot.Store, error) {
// 	// create bucket for store
// 	if err := db.Update(func(tx *bbolt.Tx) error {
// 		_, err := tx.CreateBucketIfNotExists([]byte(prefix))
// 		return err
// 	}); err != nil {
// 		return nil, err
// 	}

// 	return &BStore{
// 		prefix: prefix,
// 		db:     db,
// 	}, nil
// }

// // Visited
// func (bs *BStore) Visited(link string) bool {
// 	sum := sha256.Sum224([]byte(link))

// 	//
// 	key := strings.Join([]string{
// 		bs.prefix,
// 		hex.EncodeToString(sum[:]),
// 	}, "_")

// 	return bs.db.Update(func(tx *bbolt.Tx) error {
// 		bu := tx.Bucket([]byte(prefix))

// 		d := bu.Get([]byte(key))
// 		// if d == nil means not found
// 		if d == nil {
// 			if err := bu.Put([]byte(key), []byte(link)); err != nil {
// 				return err
// 			}
// 			return nil
// 		}

// 		return fmt.Errorf("visited")
// 	}) != nil
// }

// // Close
// func (bs *BStore) Close() error {
// 	return bs.db.Close()
// }
