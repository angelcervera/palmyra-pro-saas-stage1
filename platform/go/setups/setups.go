package setups

import (
	"log"
	"os"
)

const (
	DevCredentialsPathEnv = "FIREBASE_CONFIG"
	DevProjectEnv         = "GCLOUD_PROJECT"
)

var DevFirebasePath *string

func init() {
	devFirebasePathTmp, found := os.LookupEnv(DevCredentialsPathEnv)
	if found {
		DevFirebasePath = &devFirebasePathTmp
		log.Printf("Loading credentials at [%s]", *DevFirebasePath)
	} else {
		DevFirebasePath = nil
	}
}

// // SetHeaders sets headers and return true if it needs to go response immediately.
// func SetHeaders(w http.ResponseWriter, r *http.Request) bool {
// 	w.Header().Set("Access-Control-Allow-Origin", "*")
// 	if r.Method == http.MethodOptions {
// 		w.Header().Set("Access-Control-Allow-Methods", "*")
// 		w.Header().Set("Access-Control-Allow-Headers", "*")
// 		// w.Header().Set("Access-Control-Max-Age", "3600")
// 		w.WriteHeader(http.StatusNoContent)
// 		return true
// 	}
//
// 	return false
// }
