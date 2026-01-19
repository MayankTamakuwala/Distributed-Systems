package main

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// album is the data model for this API.
type album struct {
	ID     string  `json:"id"`
	Title  string  `json:"title"`
	Artist string  `json:"artist"`
	Price  float64 `json:"price"`
}

// albums stores data in memory for this example.
var albums = []album{
	{ID: "1", Title: "Blue Train", Artist: "John Coltrane", Price: 56.99},
	{ID: "2", Title: "Jeru", Artist: "Gerry Mulligan", Price: 17.99},
	{ID: "3", Title: "Sarah Vaughan and Clifford Brown", Artist: "Sarah Vaughan", Price: 39.99},
}

// main registers routes and starts the HTTP server.
func main() {
	router := gin.Default()
	router.GET("/albums", getAlbums)
	router.GET("/albums/:id", getAlbumByID)
	router.POST("/albums", postAlbums)
	router.PUT("/albums/:id", putAlbumByID)
	router.PATCH("/albums/:id", patchAlbumByID)
	router.DELETE("/albums/:id", deleteAlbumByID)

	router.Run("0.0.0.0:8080")
}

// getAlbums returns the full list of albums.
func getAlbums(c *gin.Context) {
	c.IndentedJSON(http.StatusOK, albums)
}

// postAlbums creates a new album from a JSON request body.
func postAlbums(c *gin.Context) {
	var newAlbum album

	if err := c.ShouldBindJSON(&newAlbum); err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"error": "invalid JSON payload"})
		return
	}

	newAlbum = normalizeAlbum(newAlbum)
	if newAlbum.ID == "" {
		// Auto-generate an ID if the client omitted one.
		newAlbum.ID = nextAlbumID()
	}
	if findAlbumIndex(newAlbum.ID) != -1 {
		c.IndentedJSON(http.StatusConflict, gin.H{"error": "album id already exists"})
		return
	}
	if err := validateAlbum(newAlbum); err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	albums = append(albums, newAlbum)
	c.IndentedJSON(http.StatusCreated, newAlbum)
}

// getAlbumByID returns a single album matching the URL id.
func getAlbumByID(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"error": "album id is required"})
		return
	}

	if index := findAlbumIndex(id); index != -1 {
		c.IndentedJSON(http.StatusOK, albums[index])
		return
	}
	c.IndentedJSON(http.StatusNotFound, gin.H{"error": "album not found"})
}

// putAlbumByID replaces an album completely (all fields required).
func putAlbumByID(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"error": "album id is required"})
		return
	}

	var input album
	if err := c.ShouldBindJSON(&input); err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"error": "invalid JSON payload"})
		return
	}

	input = normalizeAlbum(input)
	if input.ID != "" && input.ID != id {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"error": "album id in path does not match payload"})
		return
	}
	input.ID = id

	if err := validateAlbum(input); err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	index := findAlbumIndex(id)
	if index == -1 {
		c.IndentedJSON(http.StatusNotFound, gin.H{"error": "album not found"})
		return
	}

	albums[index] = input
	c.IndentedJSON(http.StatusOK, input)
}

// patchAlbumByID updates only the provided fields.
func patchAlbumByID(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"error": "album id is required"})
		return
	}

	var input album
	if err := c.ShouldBindJSON(&input); err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"error": "invalid JSON payload"})
		return
	}

	input = normalizeAlbum(input)
	if input.ID != "" && input.ID != id {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"error": "album id in path does not match payload"})
		return
	}

	index := findAlbumIndex(id)
	if index == -1 {
		c.IndentedJSON(http.StatusNotFound, gin.H{"error": "album not found"})
		return
	}

	updated := albums[index]
	if input.Title != "" {
		updated.Title = input.Title
	}
	if input.Artist != "" {
		updated.Artist = input.Artist
	}
	if input.Price > 0 {
		updated.Price = input.Price
	}

	if err := validateAlbum(updated); err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	albums[index] = updated
	c.IndentedJSON(http.StatusOK, updated)
}

// deleteAlbumByID removes an album by id.
func deleteAlbumByID(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"error": "album id is required"})
		return
	}

	index := findAlbumIndex(id)
	if index == -1 {
		c.IndentedJSON(http.StatusNotFound, gin.H{"error": "album not found"})
		return
	}

	albums = append(albums[:index], albums[index+1:]...)
	c.Status(http.StatusNoContent)
}

// validateAlbum ensures the album has all required fields.
func validateAlbum(input album) error {
	switch {
	case input.ID == "":
		return errors.New("album id is required")
	case input.Title == "":
		return errors.New("title is required")
	case input.Artist == "":
		return errors.New("artist is required")
	case input.Price <= 0:
		return errors.New("price must be greater than 0")
	}
	return nil
}

// normalizeAlbum trims whitespace to keep data consistent.
func normalizeAlbum(input album) album {
	input.ID = strings.TrimSpace(input.ID)
	input.Title = strings.TrimSpace(input.Title)
	input.Artist = strings.TrimSpace(input.Artist)
	return input
}

// findAlbumIndex searches the slice for a matching id.
func findAlbumIndex(id string) int {
	for i, a := range albums {
		if a.ID == id {
			return i
		}
	}
	return -1
}

// nextAlbumID creates a new numeric ID based on the current max.
func nextAlbumID() string {
	maxID := 0
	for _, a := range albums {
		if parsed, err := strconv.Atoi(a.ID); err == nil && parsed > maxID {
			maxID = parsed
		}
	}
	return strconv.Itoa(maxID + 1)
}
