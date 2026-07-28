package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
)

// Mengizinkan huruf, angka, spasi, dan karakter umum yang aman
var allowedPattern = regexp.MustCompile(`^[a-zA-Z0-9\s\-\_,\.\+!@#\$%^&*()\[\]{}:;'"\?\/\\~` + "`" + ` ]*$`)

// cek apakah string berisi karakter terlarang
func containsIllegal(s string) bool {
	return !allowedPattern.MatchString(s)
}

// pemeriksaan rekursif JSON
func checkJSON(v interface{}) bool {
	switch t := v.(type) {

	case string:
		return containsIllegal(t)

	case []interface{}:
		for _, item := range t {
			if checkJSON(item) {
				return true
			}
		}
		return false

	case map[string]interface{}:
		for _, val := range t {
			if checkJSON(val) {
				return true
			}
		}
		return false

	default:
		return false
	}
}

// REFACTOR ISSUE COGNITIVE COMPLEXITY - THESIS START HERE
func rejectRequest(c *gin.Context, message string) {
	c.JSON(http.StatusBadRequest, gin.H{
		"error": message,
	})
	c.Abort()
}

func validateQueryParams(c *gin.Context) bool {
	for key, vals := range c.Request.URL.Query() {

		if containsIllegal(key) {
			rejectRequest(c, "Query key mengandung karakter ilegal")
			return false
		}

		for _, v := range vals {
			if containsIllegal(v) {
				rejectRequest(c, "Query value mengandung karakter ilegal")
				return false
			}
		}
	}

	return true
}

func validateFormData(c *gin.Context) bool {
	contentType := c.GetHeader("Content-Type")

	if strings.Contains(contentType, "multipart/form-data") {
		return true
	}

	_ = c.Request.ParseMultipartForm(100 << 20)

	for key, vals := range c.Request.PostForm {

		if containsIllegal(key) {
			rejectRequest(c, "Form key mengandung karakter ilegal")
			return false
		}

		for _, v := range vals {
			if containsIllegal(v) {
				rejectRequest(c, "Form value mengandung karakter ilegal")
				return false
			}
		}
	}

	return true
}

// END THIS REFACTOR

func XSSBlocker() gin.HandlerFunc {
	return func(c *gin.Context) {
		// ===  Cek query params ===
		// START HERE REFACTOR
		if !validateQueryParams(c) {
			return
		}

		// ===  Cek form / multipart ===
		if !validateFormData(c) {
			return
		}

		// END THIS REFACTOR
		// ===  Cek JSON body ===
		ct := strings.ToLower(c.GetHeader("Content-Type"))
		if strings.Contains(ct, "application/json") {
			body, err := io.ReadAll(c.Request.Body)
			if err == nil && len(body) > 0 {
				var jsonData interface{}
				if json.Unmarshal(body, &jsonData) == nil {
					if checkJSON(jsonData) {
						c.JSON(http.StatusBadRequest, gin.H{"error": "Body JSON mengandung karakter ilegal"})
						c.Abort()
						return
					}
				}
				c.Request.Body = io.NopCloser(bytes.NewBuffer(body))
			}
		}

		c.Next()
	}
}
