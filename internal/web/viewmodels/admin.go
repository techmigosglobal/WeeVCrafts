package viewmodels

// AdminPage is the view model for the super-admin preview workspace. It is
// intentionally fixture-backed until the live admin transport is connected to
// the application services in cmd/res2.
type AdminPage struct {
	Active, Workspace, Title, Subtitle, Query, Range, Notice string
	Stats                                                    []AdminStat
	Bars                                                     []AdminBar
	Donut                                                    []AdminDonut
	Sellers                                                  []AdminSeller
	Products                                                 []AdminProduct
	Inventory                                                []AdminInventory
	Orders                                                   []AdminOrder
	Returns                                                  []AdminReturn
	Settlements                                              []AdminSettlement
	Transactions                                             []AdminTransaction
	SupportCases                                             []AdminSupportCase
	Campaigns                                                []AdminCampaign
	AuditEntries                                             []AdminAuditEntry
	Configs                                                  []AdminConfig
}

type AdminStat struct {
	Icon, Value, Label, Delta, Period string
	Positive                          bool
}

type AdminBar struct {
	Label, Value, Height, Color string
}

type AdminDonut struct {
	Label, Value, Color string
}

type AdminSeller struct {
	Name, Location, Category, Applied, Status, Image string
	Products, Orders, Revenue, Rating                string
}

type AdminProduct struct {
	Name, SKU, Seller, Category, Price, Submitted, Status, Image string
}

type AdminInventory struct {
	Name, SKU, Stock, Reorder, Reserved, Available, Location, Value, Status, Image string
}

type AdminOrder struct {
	Number, Date, Customer, Items, Value, Type, Destination, Fulfillment, Delivery, RTO string
}

type AdminReturn struct {
	Number, Customer, Product, Reason, Date, Status, Amount, Image string
}

type AdminSettlement struct {
	Seller, Orders, Gross, Commission, Amount, Status, Date, Image string
}

type AdminTransaction struct {
	Date, Type, Reference, Description, Amount, Status string
}

type AdminSupportCase struct {
	Case, Customer, Subject, Priority, Date, Status string
}

type AdminCampaign struct {
	Name, Type, Start, End, Status, Channel, Reach, Revenue string
}

type AdminAuditEntry struct {
	Date, User, Action, Entity, Details, IP string
}

type AdminConfig struct {
	Icon, Name, Value, Detail string
}

// SellerAdminPage is the seller-scoped preview workspace. The mock transport
// feeds it representative data until the live seller application services are
// connected; every record in this view model belongs to Weaver's Touch.
type SellerAdminPage struct {
	Active, Workspace, Title, Subtitle, Query, Range, Tab, Notice string
	Stats                                                         []SellerAdminStat
	Products                                                      []SellerAdminProduct
	Inventory                                                     []SellerAdminInventory
	Orders                                                        []SellerAdminOrder
	Returns                                                       []SellerAdminReturn
	Ledger                                                        []SellerAdminLedger
	Settlements                                                   []SellerAdminSettlement
	AnalyticsBars                                                 []SellerAdminBar
	Team                                                          []SellerTeamMember
}

type SellerAdminStat struct {
	Icon, Value, Label, Delta, Period, Tone string
	Positive                                bool
}

type SellerAdminBar struct {
	Label, Current, Previous string
}

type SellerAdminProduct struct {
	Name, SKU, Category, Price, Variants, Stock, Status, Availability, Image string
}

type SellerAdminInventory struct {
	Name, SKU, Variant, Location, Available, Reserved, Reorder, Status, Image string
}

type SellerAdminOrder struct {
	Number, Date, Customer, Items, Value, Type, Payment, Status, Destination, Image string
}

type SellerAdminReturn struct {
	Number, Order, Customer, Product, Reason, Date, Status, Amount, Image string
}

type SellerAdminLedger struct {
	Number, Date, Sale, Commission, Adjustments, Status, Net string
}

type SellerAdminSettlement struct {
	Number, Amount, Orders, Status, Date string
}

type SellerTeamMember struct {
	Name, Email, Role, Status, LastActive, Permissions, Image string
}

// SupportPortalPage is the support-agent-scoped preview workspace. Customer
// contact and transaction fields are intentionally minimized or masked to
// reflect the PRD's support data boundary.
type SupportPortalPage struct {
	Active, Title, Subtitle, Query, Range, Tab, Notice string
	Stats                                              []SupportPortalStat
	Cases                                              []SupportPortalCase
	Conversations                                      []SupportPortalConversation
	Directory                                          []SupportPortalDirectoryEntry
	Timeline                                           []SupportPortalTimelineEntry
}

type SupportPortalStat struct {
	Icon, Value, Label, Delta, Period, Tone string
	Positive                                bool
}

type SupportPortalCase struct {
	ID, Subject, Customer, CustomerInitials, Category, Priority, Created, Status, Age, Seller, Order, Avatar, Assignee, Channel string
}

type SupportPortalConversation struct {
	Author, Initials, Role, Message, Time, Tone, Image string
}

type SupportPortalDirectoryEntry struct {
	Name, EmailPhone, Type, Location, Orders, Cases, Status, LastActive, Image string
}

type SupportPortalTimelineEntry struct {
	Time, Title, Detail, Tone string
}
