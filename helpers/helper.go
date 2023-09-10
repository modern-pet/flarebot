package helpers

import (
	"fmt"
	"time"
)

func GetJakartaDateAndTime() (string, error) {
	location, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		fmt.Println("Error loading location:", err)
		return "", err
	}

	currentTimeInJakarta := time.Now().In(location)
	return currentTimeInJakarta.String(), nil
}

func UnixToJakartaTime(unixTimestamp int64) time.Time {
	location, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		panic(err)
	}

	return time.Unix(unixTimestamp, 0).In(location)
}
