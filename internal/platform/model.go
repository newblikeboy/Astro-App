package platform

import (
	"astroapp/backend/internal/numerology"
	"astroapp/backend/internal/swisseph"
	"time"
)

type User struct {
	ID        string    `json:"id"`
	Phone     string    `json:"phone"`
	Created   time.Time `json:"created"`
	Marketing bool      `json:"marketing"`
	Deleted   bool      `json:"deleted"`
	Analytics bool      `json:"analytics"`
}
type Session struct {
	UserID  string    `json:"user_id"`
	Role    string    `json:"role"`
	CSRF    string    `json:"csrf"`
	Expires time.Time `json:"expires"`
}
type OTP struct {
	Phone    string
	Hash     string
	Expires  time.Time
	Attempts int
	Used     bool
}
type Place struct {
	ID    string  `json:"id"`
	Name  string  `json:"name"`
	Hindi string  `json:"hindi"`
	Lat   float64 `json:"lat"`
	Lon   float64 `json:"lon"`
	Zone  string  `json:"zone"`
}
type Birth struct {
	Name string `json:"name"`
	// Deprecated: accept legacy snapshots and cached requests; new calculations use Name exclusively.
	NumerologyName string `json:"numerology_name,omitempty"`
	Date           string `json:"date"`
	Time           string `json:"time"`
	PlaceID        string `json:"place_id"`
	Unknown        bool   `json:"unknown_time"`
	Offset         *int   `json:"utc_offset_minutes,omitempty"`
}
type Rule struct {
	ID         string     `json:"id"`
	Version    int        `json:"version"`
	Planet     string     `json:"planet"`
	Sign       int        `json:"sign"`
	House      int        `json:"house,omitempty"`
	LagnaSign  int        `json:"lagna_sign,omitempty"`
	Nakshatra  int        `json:"nakshatra_number,omitempty"`
	DashaLord  string     `json:"dasha_lord,omitempty"`
	Category   string     `json:"category,omitempty"`
	Topic      string     `json:"topic"`
	Hindi      string     `json:"hindi"`
	English    string     `json:"english"`
	Mantra     string     `json:"mantra"`
	Gemstone   string     `json:"gemstone"`
	Priority   int        `json:"priority"`
	Conflict   string     `json:"conflict_group"`
	Status     string     `json:"status"`
	Approver   string     `json:"approver"`
	ApprovedAt *time.Time `json:"approved_at"`
	Demo       bool       `json:"demo"`
}
type Report struct {
	Numerology  *numerology.Reading   `json:"numerology,omitempty"`
	ID          string                `json:"id"`
	UserID      string                `json:"user_id"`
	Birth       Birth                 `json:"birth"`
	Place       Place                 `json:"place"`
	Chart       *swisseph.Kundali     `json:"chart"`
	Details     *KundliExpansion      `json:"details,omitempty"`
	Navamsa     *swisseph.Navamsa     `json:"navamsa,omitempty"`
	Dasha       *swisseph.Vimshottari `json:"dasha,omitempty"`
	Checks      []PlacementCheck      `json:"checks,omitempty"`
	Transits    *TransitSnapshot      `json:"transits,omitempty"`
	Rules       []Rule                `json:"rules"`
	Limitations []string              `json:"limitations"`
	Created     time.Time             `json:"created"`
	Engine      string                `json:"engine"`
	Demo        bool                  `json:"demo"`
}
type Order struct {
	Gemstone     *GemstoneSale  `json:"gemstone,omitempty"`
	ID           string         `json:"id"`
	UserID       string         `json:"user_id"`
	Product      string         `json:"product"`
	Title        string         `json:"title"`
	Amount       int64          `json:"amount"`
	Currency     string         `json:"currency"`
	Status       string         `json:"status"`
	ProviderID   string         `json:"provider_id"`
	PaymentID    string         `json:"payment_id"`
	ReportID     string         `json:"report_id"`
	Snapshot     *Report        `json:"snapshot,omitempty"`
	BookingID    string         `json:"booking_id"`
	Created      time.Time      `json:"created"`
	PaidAt       *time.Time     `json:"paid_at"`
	Refunded     int64          `json:"refunded"`
	Fees         int64          `json:"fees"`
	Tax          int64          `json:"tax"`
	TaxMode      string         `json:"tax_mode"`
	TaxBPS       int64          `json:"tax_bps"`
	Demo         bool           `json:"demo"`
	RequestKey   string         `json:"request_key"`
	RefundAt     *time.Time     `json:"refund_at"`
	Refunds      []RefundRecord `json:"refunds"`
	RefundReason string         `json:"refund_reason,omitempty"`
}
type RefundRecord struct {
	At     time.Time `json:"at"`
	Amount int64     `json:"amount"`
	Tax    int64     `json:"tax"`
}
type Booking struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	OrderID   string    `json:"order_id"`
	Start     time.Time `json:"start"`
	End       time.Time `json:"end"`
	HoldUntil time.Time `json:"hold_until"`
	Status    string    `json:"status"`
	MeetingID string    `json:"-"`
	JoinURL   string    `json:"join_url,omitempty"`
	Version   int       `json:"version"`
	Demo      bool      `json:"demo"`
}

// Meeting IDs remain in persistence but never in API responses.
type Meeting struct {
	ID      string `json:"id"`
	JoinURL string `json:"join_url"`
}
type Job struct {
	ID       string    `json:"id"`
	Kind     string    `json:"kind"`
	Ref      string    `json:"ref"`
	Status   string    `json:"status"`
	Attempts int       `json:"attempts"`
	Next     time.Time `json:"next"`
	Lease    time.Time `json:"lease"`
	Error    string    `json:"error"`
	Version  int       `json:"version"`
	Payload  string    `json:"payload,omitempty"`
}
type Expense struct {
	ID          string `json:"id"`
	Date        string `json:"date"`
	Category    string `json:"category"`
	Description string `json:"description"`
	Amount      int64  `json:"amount"`
	Receipt     string `json:"receipt"`
	Status      string `json:"status"`
	Approver    string `json:"approver"`
}
type Distribution struct {
	ID        string `json:"id"`
	Month     string `json:"month"`
	Partner   string `json:"partner"`
	Amount    int64  `json:"amount"`
	Reference string `json:"reference"`
}
type Reconciliation struct {
	Month              string `json:"month"`
	Reserve            int64  `json:"reserve"`
	StatementReference string `json:"statement_reference"`
	Closed             bool   `json:"closed"`
	By                 string `json:"by"`
}
type Event struct {
	Name     string    `json:"name"`
	Product  string    `json:"product"`
	Source   string    `json:"source"`
	Medium   string    `json:"medium"`
	Campaign string    `json:"campaign"`
	At       time.Time `json:"at"`
}
type Audit struct {
	At     time.Time `json:"at"`
	Actor  string    `json:"actor"`
	Action string    `json:"action"`
	Ref    string    `json:"ref"`
}
type FAQ struct {
	Question string `json:"question"`
	Answer   string `json:"answer"`
}
type Content struct {
	Intro    string            `json:"intro"`
	Videos   []string          `json:"videos"`
	FAQ      []FAQ             `json:"faq"`
	Support  string            `json:"support"`
	Policies map[string]string `json:"policies"`
	Approved bool              `json:"approved"`
}
type Window struct {
	Day  int    `json:"day"`
	From string `json:"from"`
	To   string `json:"to"`
}
type Settings struct {
	Shop              ShopSettings `json:"shop"`
	PDFPrice          int64        `json:"pdf_price"`
	ConsultationPrice int64        `json:"consultation_price"`
	Duration          int          `json:"duration"`
	Buffer            int          `json:"buffer"`
	LeadHours         int          `json:"lead_hours"`
	HorizonDays       int          `json:"horizon_days"`
	ChangeHours       int          `json:"change_hours"`
	Zone              string       `json:"zone"`
	Availability      []Window     `json:"availability"`
	Blocked           []string     `json:"blocked"`
	BookingsEnabled   bool         `json:"bookings_enabled"`
	PDFEnabled        bool         `json:"pdf_enabled"`
	TaxMode           string       `json:"tax_mode"`
	TaxBPS            int64        `json:"tax_bps"`
	LicenceConfirmed  bool         `json:"licence_confirmed"`
	Content           Content      `json:"content"`
}
type State struct {
	VideoLinks      map[string]VideoLink      `json:"video_links"`
	Gemstones       map[string]*Gemstone      `json:"gemstones"`
	Users           map[string]*User          `json:"users"`
	Sessions        map[string]*Session       `json:"sessions"`
	OTPs            map[string]*OTP           `json:"otps"`
	Reports         map[string]*Report        `json:"reports"`
	Orders          map[string]*Order         `json:"orders"`
	Bookings        map[string]*Booking       `json:"bookings"`
	Meetings        map[string]Meeting        `json:"meetings"`
	Rules           map[string]Rule           `json:"rules"`
	Jobs            map[string]*Job           `json:"jobs"`
	Webhooks        map[string]bool           `json:"webhooks"`
	Expenses        map[string]*Expense       `json:"expenses"`
	Distributions   map[string]*Distribution  `json:"distributions"`
	Reconciliations map[string]Reconciliation `json:"reconciliations"`
	Events          []Event                   `json:"events"`
	Audits          []Audit                   `json:"audits"`
	Settings        Settings                  `json:"settings"`
}

// seed creates development defaults and empty entity collections for a new application state.
func seed() *State {
	// Additive collections stay empty until the merchant adds real inventory.
	s := &State{Users: map[string]*User{}, Sessions: map[string]*Session{}, OTPs: map[string]*OTP{}, Reports: map[string]*Report{}, Orders: map[string]*Order{}, Bookings: map[string]*Booking{}, Meetings: map[string]Meeting{}, Rules: map[string]Rule{}, Jobs: map[string]*Job{}, Webhooks: map[string]bool{}, Expenses: map[string]*Expense{}, Distributions: map[string]*Distribution{}, Reconciliations: map[string]Reconciliation{}, Events: []Event{}, Audits: []Audit{}}
	s.Settings = Settings{PDFPrice: 9900, ConsultationPrice: 350100, Buffer: 15, LeadHours: 24, HorizonDays: 30, ChangeHours: 24, Zone: "Asia/Kolkata", TaxMode: "unconfigured", Availability: []Window{{1, "10:00", "13:00"}, {3, "15:00", "18:00"}, {6, "10:00", "13:00"}}, Blocked: []string{}, Content: Content{Intro: "अपनी कुंडली को समझने और सोच-समझकर आगे बढ़ने की एक सहज शुरुआत। पं. राजा मिश्रा का स्वीकृत परिचय यहाँ जोड़ा जाएगा।", Videos: FeaturedVideoIDs(), Support: "संपर्क विवरण जल्द उपलब्ध होंगे।", FAQ: []FAQ{{"जन्म समय नहीं पता हो तो?", "आप विवरण दर्ज कर सकते हैं। सही समय के बिना लग्न, भाव और ग्रहों की निश्चित स्थिति नहीं दिखाई जाएगी।"}, {"उपाय PDF कैसे मिलेगा?", "सत्यापित भुगतान के बाद PDF आपके खाते में मिलेगा। आप उसे सुरक्षित रूप से डाउनलोड कर सकेंगे।"}, {"परामर्श कैसे होगा?", "उपलब्ध समय चुनने और भुगतान के बाद आपके खाते में निजी Zoom लिंक उपलब्ध होगा। अवधि की पुष्टि बुकिंग शुरू होने से पहले की जाएगी।"}, {"बुकिंग बदल या रद्द कर सकते हैं?", "खाते से अनुरोध करें। लागू समय-सीमा और स्वीकृत रद्दीकरण नीति बुकिंग से पहले दिखाई जाएगी।"}}, Policies: map[string]string{"privacy": "समीक्षा हेतु प्रारूप: जन्म विवरण और संपर्क जानकारी केवल रिपोर्ट, खाते और बुकिंग के लिए सुरक्षित रखी जाती है। अपने खाते से मिटाने का अनुरोध करें। भुगतान अभिलेखों की कानूनी अवधि संचालक द्वारा तय की जानी है।", "terms": "समीक्षा हेतु प्रारूप: ज्योतिषीय व्याख्या सांस्कृतिक और व्यक्तिगत चिंतन के लिए है। यह चिकित्सा, कानूनी या वित्तीय सलाह नहीं है। किसी परिणाम की गारंटी नहीं है।", "refund": "समीक्षा हेतु प्रारूप: PDF और परामर्श की रद्दीकरण, पुनर्निर्धारण और धनवापसी शर्तों को व्यवसाय की स्वीकृति के बाद यहाँ प्रकाशित किया जाएगा। लाइव बिक्री अभी बंद है।", "contact": "स्वीकृत संपर्क विवरण अभी उपलब्ध नहीं हैं। कृपया लाइव सेवा शुरू होने पर यहाँ दी गई सहायता जानकारी का उपयोग करें।"}}}
	s.Rules["demo-reflection:1"] = Rule{ID: "demo-reflection", Version: 1, Topic: "चिंतन", Hindi: "यह केवल डेमो सामग्री है: अपनी प्राथमिकताओं को लिखने और नियमित आत्मचिंतन के लिए समय निकालें। व्यक्तिगत व्याख्या गुरुजी की स्वीकृति के बाद उपलब्ध होगी।", Status: "draft", Priority: 1, Demo: true}
	s.Gemstones = map[string]*Gemstone{}
	s.VideoLinks = map[string]VideoLink{}
	s.Settings.Shop = ShopSettings{TaxMode: "unconfigured"}
	return s
}
