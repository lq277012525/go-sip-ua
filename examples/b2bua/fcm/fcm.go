package fcm

import (
	"fmt"

	"context"

	firebase "firebase.google.com/go"
	"firebase.google.com/go/messaging"

	"google.golang.org/api/option"
)

func FCMPush(fcmCert string, token string, payload map[string]string) (string, error) {
	opt := option.WithCredentialsFile(fcmCert)
	app, err := firebase.NewApp(context.Background(), nil, opt)
	if err != nil {
		fmt.Println(err)
	}

	// Obtain a messaging.Client from the App.
	ctx := context.Background()
	client, err := app.Messaging(ctx)

	// See documentation on defining a message payload.
	message := &messaging.Message{
		Data:  payload,
		Token: token,
	}

	// Send a message to the device corresponding to the provided
	// registration token.
	response, err := client.Send(ctx, message)
	if err != nil {
		fmt.Println(err)
		return "", err
	}
	// Response is a message ID string.
	fmt.Println("Successfully sent message:", response)
	return response, err
}
