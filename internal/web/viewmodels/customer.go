package viewmodels

import "time"

type Product struct {
	Slug, Name, Seller, Location, Category, ImageURL, Badge string
	Price, CompareAt                                        string
	Rating                                                  string
	Reviews                                                 int
	InStock                                                 bool
}

type Seller struct {
	Slug, Name, Location, ImageURL, Description string
	Rating                                      string
	Products                                    int
}

type Category struct {
	Slug, Name, Description, ImageURL string
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

type Return struct {
	ID, Status, Reason, Requested, ProductName, ImageURL, OrderNumber string
	Amount                                                            string
}

type Address struct {
	Name, Lines, Phone string
}

type CustomerPage struct {
	Route, Title, Query, Category, Sort, Notice string
	Brand, Tagline, Description                 string
	Products                                    []Product
	Categories                                  []Category
	Sellers                                     []Seller
	Cart                                        []CartItem
	Orders                                      []Order
	Returns                                     []Return
	Address                                     Address
	CartCount, WishlistCount                    int
	Page, Pages                                 int
	CheckoutStep                                int
	Authenticated, Mock                         bool
	Now                                         time.Time
}
