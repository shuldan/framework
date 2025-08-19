package main

import (
	"context"
	"fmt"
	"github.com/shuldan/framework/pkg/events"
	"log"
)

type UserEvent struct {
	Username string
	Action   string
}

type PostEvent struct {
	Id   int
	Text string
}

type UserEventListener struct {
}

func (u *UserEventListener) Handle(ctx context.Context, e UserEvent) error {
	fmt.Printf("User %s did %s\n", e.Username, e.Action)
	return nil
}

func main() {
	bus := events.New()

	// Подписка
	_ = bus.Subscribe(
		(*UserEvent)(nil), // маркер типа
		&UserEventListener{},
	)

	// Публикация
	_ = bus.Publish(context.Background(), UserEvent{
		Username: "alice",
		Action:   "login",
	})

	_ = bus.Publish(context.Background(), PostEvent{
		Id:   1,
		Text: "Hello world",
	})

	log.Println("published")

	_ = bus.Close()
}
