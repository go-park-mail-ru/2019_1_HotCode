package storage

import "github.com/go-redis/redis"

var Client *redis.Client

// Connect открывает соединение с хранилищем для sessions
func Connect(storageUser, storagePass, storageHost string) error {
	var err error
	Client = redis.NewClient(&redis.Options{
		Addr:     storageHost,
		Password: storagePass,
		DB:       0,
	})
	_, err = Client.Ping().Result()
	if err != nil {
		return err
	}

	return nil
}

func Close() error {
	return Client.Close()
}
