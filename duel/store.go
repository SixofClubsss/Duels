package duel

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/dReam-dApps/dReams/menu"
	"go.etcd.io/bbolt"
)

// Store duel index in boltdb, if using gravdb it will not store index
func storeIndex() (err error) {
	if menu.Gnomes.DBType != "boltdb" {
		return
	}

	if menu.Gnomes.Indexer == nil {
		logger.Errorln("[storeIndex] DB is nil")
		return
	}

	db := menu.Gnomes.Indexer.BBSBackend.DB
	for menu.Gnomes.IsWriting() {
		time.Sleep(20 * time.Millisecond)
		logger.Warnln("[storeIndex] write wait")
	}

	menu.Gnomes.Writing(true)

	bucket := "DUELBUCKET"

	err = db.Update(func(tx *bbolt.Tx) (err error) {

		b, err := tx.CreateBucketIfNotExists([]byte(bucket))
		if err != nil {
			logger.Errorf("[storeIndex] err creating bucket %s\n", err)
			return
		}

		key := "DUELS"

		mar, err := json.Marshal(&Duels)
		if err != nil {
			logger.Errorln("[storeIndex]", key, mar, bucket, err)
			return
		}

		err = b.Put([]byte(key), []byte(mar))
		if err != nil {
			logger.Errorln("[storeIndex]", key, mar, bucket, err)
			return
		}

		return
	})

	menu.Gnomes.Writing(false)

	return
}

// Get duel index from boltdb
func getIndex() (stored entries) {
	stored.Index = make(map[uint64]entry)
	if menu.Gnomes.DBType != "boltdb" {
		return
	}

	if menu.Gnomes.Indexer == nil {
		logger.Errorln("[getIndex] DB is nil")
		return
	}

	db := menu.Gnomes.Indexer.BBSBackend.DB
	bucket := "DUELBUCKET"
	key := "DUELS"

	db.View(func(tx *bbolt.Tx) error {
		if b := tx.Bucket([]byte(bucket)); b != nil {
			if ok := b.Get([]byte(key)); ok != nil {
				err := json.Unmarshal(ok, &stored)
				if err != nil {
					logger.Errorln("[getIndex]", err)
				}
				return nil
			}
			logger.Warnln("[getIndex] Error - key is nil")
		}
		return nil
	})

	return
}

// Delete stored duel index from boltdb
func deleteIndex() {
	if menu.Gnomes.DBType != "boltdb" {
		return
	}

	if menu.Gnomes.Indexer == nil {
		logger.Errorln("[deleteIndex] DB is nil")
		return
	}

	db := menu.Gnomes.Indexer.BBSBackend.DB
	bucket := "DUELBUCKET"
	err := db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		return b.Delete([]byte("DUELS"))
	})

	if err != nil {
		logger.Errorln("[deleteIndex]", bucket, err)
		return
	}

	logger.Println("[deleteIndex] DUELS", bucket, "Deleted")

}

// Download NFA SCID icon image file as []byte
func downloadBytes(scid string) ([]byte, error) {
	client := http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(menu.GetAssetUrl(1, scid))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	image, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return image, nil
}
