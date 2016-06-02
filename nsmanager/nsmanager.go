package nsmanager

import (
	"errors"
	"fmt"
	"gopkg.in/redis.v3"
)

func Query(queryType, recordType, queryAttribute string, redisClient *redis.Client) (string, error) {
	searchString := fmt.Sprintf("%s:%s:%s", queryType, recordType, queryAttribute)
	val, err := redisClient.Get(searchString).Result()
	if err != nil {
		return "", errors.New("string not found.")
	} else {
		return val, nil
	}
	return "", nil
}
