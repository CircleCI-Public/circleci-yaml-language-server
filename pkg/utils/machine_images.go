package utils

import (
	"fmt"
	"strings"
	"time"
)

var ValidARMOrMachineImagesUbuntu2004 = []string{
	// Ubuntu 2004
	"ubuntu-2004:current",
	"ubuntu-2004:edge",
	"ubuntu-2004:2024.04.4",
	"ubuntu-2004:2024.01.2",
	"ubuntu-2004:2024.01.1",
	"ubuntu-2004:2023.10.1",
	"ubuntu-2004:2023.07.1",
	"ubuntu-2004:2023.04.2",
	"ubuntu-2004:2023.04.1",
	"ubuntu-2004:2023.02.1",
	"ubuntu-2004:2022.10.1",
	"ubuntu-2004:2022.07.1",
	"ubuntu-2004:2022.04.2",
	"ubuntu-2004:2022.04.1",
	"ubuntu-2004:202201-02",
	"ubuntu-2004:202201-01",
	"ubuntu-2004:202111-02",
	"ubuntu-2004:202111-01",
	"ubuntu-2004:202107-02",
	"ubuntu-2004:202104-01",
	"ubuntu-2004:202101-01",
	"ubuntu-2004:202010-01",
}
var ValidARMOrMachineImagesUbuntu2204 = []string{
	// Ubuntu 2204
	"ubuntu-2204:current",
	"ubuntu-2204:edge",
	"ubuntu-2204:2024.04.4",
	"ubuntu-2204:2024.01.2",
	"ubuntu-2204:2024.01.1",
	"ubuntu-2204:2023.10.1",
	"ubuntu-2204:2023.07.2",
	"ubuntu-2204:2023.04.2",
	"ubuntu-2204:2023.04.1",
	"ubuntu-2204:2023.02.1",
	"ubuntu-2204:2022.10.2",
	"ubuntu-2204:2022.10.1",
	"ubuntu-2204:2022.07.2",
	"ubuntu-2204:2022.07.1",
	"ubuntu-2204:2022.04.2",
	"ubuntu-2204:2022.04.1",
}
var ValidARMOrMachineImages = append(ValidARMOrMachineImagesUbuntu2004, ValidARMOrMachineImagesUbuntu2204...)

func GetLatestUbuntu2204Image() string {
	latestImg := ValidARMOrMachineImagesUbuntu2204[0]
	for _, img := range ValidARMOrMachineImagesUbuntu2204 {
		latestImg = getMoreRecentImg(img, latestImg)
	}
	return latestImg
}

func getMoreRecentImg(img1 string, img2 string) string {
	img1Version, err1 := getImgVersion(img1)
	if err1 != nil {
		return img2
	}
	img2Version, err2 := getImgVersion(img2)

	if err2 != nil {
		return img1
	}
	img1Date, timeErr := time.Parse("2006.01.2", img1Version)

	if timeErr != nil {
		return img2
	}

	img2Date, timeErr2 := time.Parse("2006.01.2", img2Version)

	if timeErr2 != nil {
		return img1
	}
	if img1Date.Before(img2Date) {
		return img2
	}

	return img1
}

func getImgVersion(img string) (string, error) {
	tokenizedImg := strings.Split(img, ":")
	if len(tokenizedImg) == 2 {
		return tokenizedImg[1], nil
	}
	return "", fmt.Errorf("Image version not supported")
}
