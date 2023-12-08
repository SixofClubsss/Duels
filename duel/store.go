package duel

import (
	"io"
	"net/http"
	"time"

	"github.com/dReam-dApps/dReams/gnomes"
)

// Download NFA SCID icon image file as []byte
func downloadBytes(scid string) ([]byte, error) {
	client := http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(findCollectionURL(scid))
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

// Find which images should be used for a collection, icon or main file
func findCollectionURL(scid string) (url string) {
	if gnomon.IsReady() {
		w := 1
		coll, _ := gnomon.GetSCIDValuesByKey(scid, "collection")
		if coll != nil {
			switch coll[0] {
			case "High Strangeness":
				w = 0
			default:
				// nothing, w 1 is icon
			}

			url = gnomes.GetAssetUrl(w, scid)
		}
	}

	return
}
