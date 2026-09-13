package handlers

import "github.com/wecratfs/commerce/internal/web/viewmodels"

var mockProducts = []viewmodels.Product{
	{Slug: "madhubani-tree", Name: "Madhubani Painting - Tree of Life", Seller: "Mithila Arts", Location: "Madhubani, Bihar", Category: "Arts", ImageURL: "/assets/images/customer/madhubani-tree.png", Badge: "Bestseller", Price: "INR 2,499", CompareAt: "INR 3,500", Rating: "4.8", Reviews: 124, InStock: true},
	{Slug: "brass-elephant", Name: "Brass Elephant Figurine", Seller: "Kalakriti", Location: "Moradabad, Uttar Pradesh", Category: "Crafts", ImageURL: "/assets/images/customer/brass-elephant.png", Badge: "Handmade", Price: "INR 1,899", CompareAt: "INR 2,400", Rating: "4.7", Reviews: 98, InStock: true},
	{Slug: "chanderi-royal", Name: "Chanderi Silk Cotton Saree - Royal Maroon", Seller: "Weaver's Touch", Location: "Maheshwar, Madhya Pradesh", Category: "Sarees", ImageURL: "/assets/images/customer/saree-maroon.png", Badge: "Handloom", Price: "INR 5,999", CompareAt: "INR 7,999", Rating: "4.8", Reviews: 218, InStock: true},
	{Slug: "wooden-box", Name: "Carved Wooden Jewellery Box", Seller: "Artisans of Rajasthan", Location: "Jaipur, Rajasthan", Category: "Home & Living", ImageURL: "/assets/images/customer/wooden-box.png", Badge: "Made in India", Price: "INR 5,299", CompareAt: "INR 6,499", Rating: "4.8", Reviews: 176, InStock: true},
	{Slug: "ceramic-mugs", Name: "Handpainted Ceramic Mugs (Set of 2)", Seller: "Clay & Co.", Location: "Khurja, Uttar Pradesh", Category: "Home & Living", ImageURL: "/assets/images/customer/ceramic-mugs.png", Badge: "Eco Friendly", Price: "INR 1,299", CompareAt: "INR 1,799", Rating: "4.5", Reviews: 91, InStock: true},
	{Slug: "blue-tote", Name: "Ikat Handwoven Tote Bag", Seller: "Loom & Life", Location: "Hyderabad, Telangana", Category: "Fashion", ImageURL: "/assets/images/customer/blue-tote.png", Badge: "New", Price: "INR 1,299", CompareAt: "", Rating: "4.5", Reviews: 61, InStock: true},
	{Slug: "blue-silk", Name: "Maheshwari Handloom Cotton Silk Saree", Seller: "Mithila Arts", Location: "Madhubani, Bihar", Category: "Sarees", ImageURL: "/assets/images/customer/saree-blue.png", Badge: "Bestseller", Price: "INR 4,299", CompareAt: "INR 5,699", Rating: "4.7", Reviews: 89, InStock: true},
	{Slug: "terracotta-lamp", Name: "Terracotta Diya Set (Set of 6)", Seller: "Sacred Spaces", Location: "Khurja, Uttar Pradesh", Category: "Gifts", ImageURL: "/assets/images/customer/terracotta-lamp.png", Badge: "Festive Edit", Price: "INR 899", CompareAt: "", Rating: "4.6", Reviews: 98, InStock: true},
	{Slug: "terracotta-planter", Name: "Terracotta Planter", Seller: "WeeVCrafts", Location: "Bengaluru, Karnataka", Category: "Home & Living", ImageURL: "/assets/images/customer/terracotta-planter.png", Badge: "Homegrown Decor", Price: "INR 1,099", CompareAt: "", Rating: "4.4", Reviews: 73, InStock: true},
	{Slug: "black-cushion", Name: "Block Print Cushion Cover", Seller: "The Rustic Home", Location: "Udaipur, Rajasthan", Category: "Home & Living", ImageURL: "/assets/images/customer/black-cushion.png", Badge: "Handloom Classics", Price: "INR 799", CompareAt: "", Rating: "4.3", Reviews: 55, InStock: true},
}

var mockCategories = []viewmodels.Category{
	{Slug: "arts", Name: "Arts", Description: "Paintings, sculptures, traditional art & more", ImageURL: "/assets/images/customer/madhubani-tree.png"},
	{Slug: "crafts", Name: "Crafts", Description: "Home decor, handicrafts, lifestyle & gifts", ImageURL: "/assets/images/customer/wooden-box.png"},
	{Slug: "sarees", Name: "Sarees", Description: "Handloom, heritage & contemporary", ImageURL: "/assets/images/customer/saree-maroon.png"},
	{Slug: "home-living", Name: "Home & Living", Description: "Thoughtful objects for every room", ImageURL: "/assets/images/customer/ceramic-mugs.png"},
	{Slug: "fashion", Name: "Fashion & Accessories", Description: "Made slowly, made beautifully", ImageURL: "/assets/images/customer/blue-tote.png"},
	{Slug: "gifts", Name: "Gifts & Occasions", Description: "Meaningful gifts for every moment", ImageURL: "/assets/images/customer/terracotta-lamp.png"},
}

var mockSellers = []viewmodels.Seller{
	{Slug: "mithila-arts", Name: "Mithila Arts", Location: "Madhubani, Bihar", ImageURL: "/assets/images/customer/maker-mithila.png", Description: "Hand-painted stories from Bihar. Every piece is made by rural women artisans keeping a living tradition bright.", Rating: "4.8", Products: 120},
	{Slug: "clay-and-co", Name: "Clay & Co.", Location: "Khurja, Uttar Pradesh", ImageURL: "/assets/images/customer/ceramic-mugs.png", Description: "Earthy objects shaped slowly in the pottery town of Khurja.", Rating: "4.6", Products: 89},
	{Slug: "weavers-touch", Name: "Weaver's Touch", Location: "Maheshwar, Madhya Pradesh", ImageURL: "/assets/images/customer/saree-blue.png", Description: "Traditional handlooms for a brighter, more thoughtful home.", Rating: "4.8", Products: 142},
	{Slug: "rustic-home", Name: "The Rustic Home", Location: "Udaipur, Rajasthan", ImageURL: "/assets/images/customer/wooden-box.png", Description: "A home for useful, enduring craft.", Rating: "4.5", Products: 76},
}

func productBySlug(slug string) (viewmodels.Product, bool) {
	for _, product := range mockProducts {
		if product.Slug == slug {
			return product, true
		}
	}
	return viewmodels.Product{}, false
}

func sellerBySlug(slug string) (viewmodels.Seller, bool) {
	for _, seller := range mockSellers {
		if seller.Slug == slug {
			return seller, true
		}
	}
	return viewmodels.Seller{}, false
}

func seedCart() []viewmodels.CartItem {
	return []viewmodels.CartItem{{Product: mockProducts[2], Quantity: 1, Delivery: "4 - 6 days"}, {Product: mockProducts[0], Quantity: 1, Delivery: "4 - 6 days"}, {Product: mockProducts[1], Quantity: 1, Delivery: "4 - 6 days"}}
}

func seedOrders() []viewmodels.Order {
	return []viewmodels.Order{
		{Number: "#WC2504267819", Status: "Processing", Payment: "Paid via UPI", Date: "26 Apr 2024", Total: "INR 14,797", Items: seedCart()},
		{Number: "#WC2504216632", Status: "Delivered", Payment: "Paid via Credit Card", Date: "21 Apr 2024", Total: "INR 2,499", Items: []viewmodels.CartItem{{Product: mockProducts[0], Quantity: 1}}},
		{Number: "#WC2504149981", Status: "Delivered", Payment: "Paid via UPI", Date: "14 Apr 2024", Total: "INR 8,298", Items: []viewmodels.CartItem{{Product: mockProducts[2], Quantity: 1}}},
		{Number: "#WC2504097712", Status: "Cancelled", Payment: "Paid via Net Banking", Date: "09 Apr 2024", Total: "INR 1,899", Items: []viewmodels.CartItem{{Product: mockProducts[1], Quantity: 1}}},
	}
}

func mockReturns() []viewmodels.Return {
	return []viewmodels.Return{
		{ID: "RET123456", Status: "Return in Progress", Reason: "Received a different color than ordered.", Requested: "12 Nov 2024", ProductName: mockProducts[2].Name, ImageURL: mockProducts[2].ImageURL, OrderNumber: "#WC98765432", Amount: "INR 5,999"},
		{ID: "RET987654", Status: "Under Review", Reason: "Product arrived damaged.", Requested: "10 Nov 2024", ProductName: mockProducts[4].Name, ImageURL: mockProducts[4].ImageURL, OrderNumber: "#WC87654321", Amount: "INR 1,299"},
		{ID: "RET765432", Status: "Refund Completed", Reason: "Changed my mind.", Requested: "02 Nov 2024", ProductName: mockProducts[1].Name, ImageURL: mockProducts[1].ImageURL, OrderNumber: "#WC76543210", Amount: "INR 1,899"},
	}
}
