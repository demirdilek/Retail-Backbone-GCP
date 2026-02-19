package database

import "github.com/google/uuid"

// CartItem represents an item while it's in the user's hand
type CartItem struct {
	Product  Product
	Quantity int
}

// Cart represents the current shopping session
type Cart struct {
	ID    uuid.UUID
	Items []CartItem
}

// Total calculates the sum of the cart
func (c *Cart) Total() float64 {
	var total float64
	for _, item := range c.Items {
		total += item.Product.PriceEuro * float64(item.Quantity)
	}
	return total
}
