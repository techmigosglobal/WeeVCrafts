package viewmodels

import (
	"net/url"
	"time"
)

type Product struct {
	Slug, Name, Seller, Location, Category, ImageURL, ImageSrcSet, Badge string
	Price, CompareAt, Discount                                           string
	Rating                                                               string
	Reviews                                                              int
	InStock                                                              bool
}

type Seller struct {
	Slug, Name, Location, ImageURL, Description string
	Rating                                      string
	Products                                    int
}

type Brand struct {
	Slug, Name, Tagline, Location, ImageURL, Description string
	Products                                             int
}

type Category struct {
	Slug, Name, Description, ImageURL string
	Products                          int
}

type CartItem struct {
	Product  Product
	Quantity int
	Delivery string
}

type Order struct {
	Number, Status, Payment, Date string
	Total                         string
	Items                         []CartItem
}

type PaymentActivity struct {
	Reference, Amount, Date, Method, OrderNumber, Status string
}

type Return struct {
	ID, Status, Reason, Requested, ProductName, ImageURL, OrderNumber string
	Amount                                                            string
}

type Address struct {
	Name, Lines, Phone string
}

type CustomerPage struct {
	Route, Title, Query, Category, Sort, Notice     string
	Brand, Tagline, Description                     string
	Filters                                         url.Values
	Product                                         Product
	SavedProducts                                   []Product
	ItemsTotal, ShippingTotal, DiscountTotal, Total string
	Products                                        []Product
	FilterCounts                                    map[string]int
	Categories                                      []Category
	Sellers                                         []Seller
	Brands                                          []Brand
	Cart                                            []CartItem
	Orders                                          []Order
	PaymentActivity                                 []PaymentActivity
	SelectedOrder                                   Order
	Returns                                         []Return
	Address                                         Address
	CartCount, WishlistCount                        int
	Page, Pages                                     int
	CheckoutStep                                    int
	Authenticated, Mock                             bool
	Now                                             time.Time
}
