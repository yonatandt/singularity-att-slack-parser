package utils

import (
	"fmt"
	"strings"
)

func ReplaceCommasWithQuotationMarks(s string) string {
	s = strings.Replace(s, "”", "\"", -1)
	s = strings.Replace(s, "“", "\"", -1)
	return s
}

func ParseAndFixQueryInMessage(s string) string {
	s = strings.Replace(s, "\n", " ", -1)
	arr := strings.Split(s, "&lt;query&gt;")
	if len(arr) < 2 {
		return s
	}
	arr[1] = strings.Replace(arr[1], "\"", "\\\"", -1)
	return strings.Join(arr, "")
}

func RemoveFirstCharAndLastChar(str string) string {
	if len(str) < 2 {
		return str
	}
	str = strings.Split(str, "|")[0]
	return str[1 : len(str)-1]
}

func TranslateToBoolean(str string) bool {
	return str == "Yes"
}

func GetRepoNameFromGithubPrURL(url string) string {
	splitArr := strings.Split(url, "/")
	if len(splitArr) < 5 {
		return "attribution"
	}
	return fmt.Sprintf("%s/%s", splitArr[3], splitArr[4])
}
