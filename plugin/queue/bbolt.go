package queue

// var (
// 	ErrEmptyQueue = errors.New("queue is empty")
// )

// // prefix
// var prefix = "queue"

// // BQueue
// type BQueue struct {
// 	prefix string
// 	db     *bbolt.DB // Bolt stores its keys in byte-sorted order within a bucket.
// }

// // NewBQueue
// func NewBQueue(db *bbolt.DB) (wbot.Queue, error) {
// 	if err := db.Update(func(tx *bbolt.Tx) error {
// 		_, err := tx.CreateBucketIfNotExists([]byte(prefix))
// 		return err
// 	}); err != nil {
// 		return nil, err
// 	}

// 	return &BQueue{
// 		prefix: prefix,
// 		db:     db,
// 	}, nil
// }

// // Enqueue
// func (bq *BQueue) Enqueue(req wbot.Request) error {
// 	var buf bytes.Buffer
// 	if err := gob.NewEncoder(&buf).Encode(req); err != nil {
// 		return err
// 	}

// 	return bq.db.Update(func(tx *bbolt.Tx) error {
// 		bu := tx.Bucket([]byte(prefix))

// 		var key = make([]byte, 8)
// 		seq, err := bu.NextSequence()
// 		if err != nil {
// 			return err
// 		}

// 		binary.BigEndian.PutUint64(key, seq)

// 		return bu.Put(key, buf.Bytes())
// 	})
// }

// // Dequeue
// func (bq *BQueue) Dequeue() (wbot.Request, error) {
// 	// get from db
// 	var req wbot.Request
// 	if err := bq.db.Update(func(tx *bbolt.Tx) error {
// 		bu := tx.Bucket([]byte(prefix))

// 		c := bu.Cursor()

// 		k, v := c.First()
// 		if k == nil {
// 			return ErrEmptyQueue
// 		}

// 		if err := gob.NewDecoder(bytes.NewReader(v)).Decode(&req); err != nil {
// 			return err
// 		}

// 		return c.Delete()
// 	}); err != nil {
// 		return wbot.Request{}, err
// 	}

// 	return req, nil
// }

// // Next
// func (bq *BQueue) Next() bool {
// 	return bq.db.View(func(tx *bbolt.Tx) error {
// 		bu := tx.Bucket([]byte(bq.prefix))

// 		c := bu.Cursor()

// 		k, _ := c.First()
// 		if k == nil {
// 			return ErrEmptyQueue
// 		}

// 		return nil
// 	}) == nil
// }

// // Close
// func (bq *BQueue) Close() error {
// 	return bq.db.Close()
// }
