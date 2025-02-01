package main

import (
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Receipt struct {
	Retailer     string `json:"retailer"`
	PurchaseDate string `json:"purchaseDate"`
	PurchaseTime string `json:"purchaseTime"`
	Items        []Item `json:"items"`
	Total        string `json:"total"`
}

type Item struct {
	ShortDescription string `json:"shortDescription"`
	Price            string `json:"price"`
}

type ReceiptStore struct {
	mu       sync.Mutex
	receipts map[string]int
}

func NewReceiptStore() *ReceiptStore {
	return &ReceiptStore{
		receipts: make(map[string]int),
	}
}

func (s *ReceiptStore) AddReceipt(receipt Receipt) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Generate unique ID
	id := uuid.New().String()

	// Calculate points
	points := calculatePoints(receipt)

	// Store receipt and points
	s.receipts[id] = points

	return id
}

func (s *ReceiptStore) GetPoints(id string) (int, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	points, exists := s.receipts[id]
	return points, exists
}

func calculatePoints(receipt Receipt) int {
	points := 0

	// Rule 1: One point for every alphanumeric character in the retailer name
	for _, char := range receipt.Retailer {
		if (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9') {
			points++
		}
	}

	// Rule 2: 50 points if the total is a round dollar amount with no cents
	if receipt.Total[len(receipt.Total)-3:] == ".00" {
		points += 50
	}

	// Rule 3: 25 points if the total is a multiple of 0.25
	if totalFloat, err := strconv.ParseFloat(receipt.Total, 64); err == nil && int(totalFloat*100)%25 == 0 {
		points += 25
	}

	// Rule 4: 5 points for every two items on the receipt
	points += (len(receipt.Items) / 2) * 5

	// Rule 5: If the trimmed length of the item description is a multiple of 3, multiply the price by 0.2 and round up
	for _, item := range receipt.Items {
		trimmedLength := len(strings.TrimSpace(item.ShortDescription))
		if trimmedLength%3 == 0 {
			if price, err := strconv.ParseFloat(item.Price, 64); err == nil {
				points += int(math.Ceil(price * 0.2))
			}
		}
	}

	// Rule 6: 6 points if the purchase date is odd
	if dateParts := strings.Split(receipt.PurchaseDate, "-"); len(dateParts) == 3 {
		if day, err := strconv.Atoi(dateParts[2]); err == nil && day%2 == 1 {
			points += 6
		}
	}

	// Rule 7: 10 points if the purchase time is between 2:00 PM and 4:00 PM
	if timeParts := strings.Split(receipt.PurchaseTime, ":"); len(timeParts) == 2 {
		if hour, err := strconv.Atoi(timeParts[0]); err == nil && hour >= 14 && hour < 16 {
			points += 10
		}
	}

	return points
}

func main() {
	receiptStore := NewReceiptStore()
	r := gin.Default()

	r.POST("/receipts/process", func(c *gin.Context) {
		var receipt Receipt
		if err := c.ShouldBindJSON(&receipt); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
			return
		}

		id := receiptStore.AddReceipt(receipt)
		c.JSON(http.StatusOK, gin.H{"id": id})
	})

	r.GET("/receipts/:id/points", func(c *gin.Context) {
		id := c.Param("id")
		if points, exists := receiptStore.GetPoints(id); exists {
			c.JSON(http.StatusOK, gin.H{"points": points})
		} else {
			c.JSON(http.StatusNotFound, gin.H{"error": "Receipt not found"})
		}
	})

	r.Run(":8080")
}
