package http

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
)

type HTTPHandlers struct{}

func NewHTTPHandlers() *HTTPHandlers {
	return &HTTPHandlers{}
}

// Health check endpoint
func (h *HTTPHandlers) Health(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]interface{}{
		"status": "healthy",
		"timestamp": time.Now().Unix(),
	})
}

// Echo back request headers and body
func (h *HTTPHandlers) EchoGet(c echo.Context) error {
	headers := make(map[string]string)
	for key, values := range c.Request().Header {
		if len(values) > 0 {
			headers[key] = values[0]
		}
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"method": c.Request().Method,
		"path": c.Path(),
		"query": c.QueryParams(),
		"headers": headers,
		"timestamp": time.Now().Unix(),
	})
}

// Echo back JSON payload with graceful error handling
func (h *HTTPHandlers) EchoPost(c echo.Context) error {
	headers := make(map[string]string)
	for key, values := range c.Request().Header {
		if len(values) > 0 {
			headers[key] = values[0]
		}
	}

	// Read the raw body first
	bodyBytes, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Failed to read request body",
			"details": err.Error(),
			"timestamp": time.Now().Unix(),
		})
	}

	bodyString := string(bodyBytes)
	
	// Response structure
	response := map[string]interface{}{
		"method": c.Request().Method,
		"path": c.Path(),
		"headers": headers,
		"timestamp": time.Now().Unix(),
	}

	// Check if body is empty
	if len(strings.TrimSpace(bodyString)) == 0 {
		response["body"] = nil
		response["body_raw"] = ""
		return c.JSON(http.StatusOK, response)
	}

	// Try to parse as JSON
	var jsonBody interface{}
	if err := json.Unmarshal(bodyBytes, &jsonBody); err != nil {
		// JSON parsing failed - return graceful error with raw body
		response["body"] = nil
		response["body_raw"] = bodyString
		response["json_parse_error"] = map[string]interface{}{
			"error": "Invalid JSON format",
			"details": err.Error(),
			"position": getJSONErrorPosition(err),
		}
		return c.JSON(http.StatusOK, response) // Still return 200 for debugging
	}

	// JSON parsing succeeded
	response["body"] = jsonBody
	response["body_raw"] = bodyString
	return c.JSON(http.StatusOK, response)
}

// Helper function to extract position information from JSON errors
func getJSONErrorPosition(err error) string {
	errStr := err.Error()
	// Extract position info from common JSON errors
	if strings.Contains(errStr, "character") {
		return errStr
	}
	if strings.Contains(errStr, "offset") {
		return errStr
	}
	return "Unknown position"
}

// Delayed response for timeout testing
func (h *HTTPHandlers) Delay(c echo.Context) error {
	secondsStr := c.Param("seconds")
	seconds, err := strconv.Atoi(secondsStr)
	if err != nil || seconds < 0 || seconds > 30 {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Invalid delay parameter. Must be 0-30 seconds",
			"provided": secondsStr,
			"timestamp": time.Now().Unix(),
		})
	}

	time.Sleep(time.Duration(seconds) * time.Second)

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "Response after delay",
		"delay_seconds": seconds,
		"timestamp": time.Now().Unix(),
	})
}

// Return specific HTTP status codes
func (h *HTTPHandlers) Status(c echo.Context) error {
	codeStr := c.Param("code")
	code, err := strconv.Atoi(codeStr)
	if err != nil || code < 100 || code > 599 {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Invalid status code. Must be 100-599",
			"provided": codeStr,
			"timestamp": time.Now().Unix(),
		})
	}

	message := http.StatusText(code)
	if message == "" {
		message = fmt.Sprintf("Status code %d", code)
	}

	return c.JSON(code, map[string]interface{}{
		"status_code": code,
		"message": message,
		"timestamp": time.Now().Unix(),
	})
}