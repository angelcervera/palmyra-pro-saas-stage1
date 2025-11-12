package gcp

import (
	"context"
	"fmt"

	firebase "firebase.google.com/go/v4"
	firebaseauth "firebase.google.com/go/v4/auth"
	"github.com/zenGate-Global/palmyra-pro-saas/platform/go/setups"
	"google.golang.org/api/option"
)

// 	firebase "firebase.google.com/go/v4"

// GetApp Creates a Firebase App instance.
func GetApp(ctx context.Context, pathToJson *string) (app *firebase.App, err error) {
	if pathToJson != nil {
		sa := option.WithCredentialsFile(*pathToJson)
		app, err = firebase.NewApp(ctx, nil, sa)
	} else {
		app, err = firebase.NewApp(ctx, nil)
	}

	if err != nil {
		return nil, err
	}
	return
}

// InitFirebaseAuth initializes the Firebase App and returns an Auth client.
// Firestore is not used in this project, so no Firestore client is created.
func InitFirebaseAuth(ctx context.Context) (*firebase.App, *firebaseauth.Client, error) {
	firebaseApp, err := GetApp(ctx, setups.DevFirebasePath)
	if err != nil {
		return nil, nil, fmt.Errorf("error initializing firebase app [%w]", err)
	}

	fbAuth, err := firebaseApp.Auth(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("error initializing firebase auth [%w]", err)
	}

	return firebaseApp, fbAuth, nil
}
