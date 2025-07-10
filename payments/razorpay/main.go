package razorpay

import (
	"encoding/json"
	"strconv"
	"time"

	"github.com/MelloB1989/karma/config"
	"github.com/razorpay/razorpay-go"
)

type Currency string

const (
	AED Currency = "AED"
	ALL Currency = "ALL"
	AMD Currency = "AMD"
	ARS Currency = "ARS"
	AUD Currency = "AUD"
	AWG Currency = "AWG"
	AZN Currency = "AZN"
	BAM Currency = "BAM"
	BBD Currency = "BBD"
	BDT Currency = "BDT"
	BGN Currency = "BGN"
	BHD Currency = "BHD"
	BIF Currency = "BIF"
	BMD Currency = "BMD"
	BND Currency = "BND"
	BOB Currency = "BOB"
	BRL Currency = "BRL"
	BSD Currency = "BSD"
	BTN Currency = "BTN"
	BWP Currency = "BWP"
	BZD Currency = "BZD"
	CAD Currency = "CAD"
	CHF Currency = "CHF"
	CLP Currency = "CLP"
	CNY Currency = "CNY"
	COP Currency = "COP"
	CRC Currency = "CRC"
	CUP Currency = "CUP"
	CVE Currency = "CVE"
	CZK Currency = "CZK"
	DJF Currency = "DJF"
	DKK Currency = "DKK"
	DOP Currency = "DOP"
	DZD Currency = "DZD"
	EGP Currency = "EGP"
	ETB Currency = "ETB"
	EUR Currency = "EUR"
	FJD Currency = "FJD"
	GBP Currency = "GBP"
	GHS Currency = "GHS"
	GIP Currency = "GIP"
	GMD Currency = "GMD"
	GNF Currency = "GNF"
	GTQ Currency = "GTQ"
	GYD Currency = "GYD"
	HKD Currency = "HKD"
	HNL Currency = "HNL"
	HRK Currency = "HRK"
	HTG Currency = "HTG"
	HUF Currency = "HUF"
	IDR Currency = "IDR"
	ILS Currency = "ILS"
	INR Currency = "INR"
	IQD Currency = "IQD"
	ISK Currency = "ISK"
	JMD Currency = "JMD"
	JOD Currency = "JOD"
	JPY Currency = "JPY"
	KES Currency = "KES"
	KGS Currency = "KGS"
	KHR Currency = "KHR"
	KMF Currency = "KMF"
	KRW Currency = "KRW"
	KWD Currency = "KWD"
	KYD Currency = "KYD"
	KZT Currency = "KZT"
	LAK Currency = "LAK"
	LKR Currency = "LKR"
	LRD Currency = "LRD"
	LSL Currency = "LSL"
	MAD Currency = "MAD"
	MDL Currency = "MDL"
	MGA Currency = "MGA"
	MKD Currency = "MKD"
	MMK Currency = "MMK"
	MNT Currency = "MNT"
	MOP Currency = "MOP"
	MUR Currency = "MUR"
	MVR Currency = "MVR"
	MWK Currency = "MWK"
	MXN Currency = "MXN"
	MYR Currency = "MYR"
	MZN Currency = "MZN"
	NAD Currency = "NAD"
	NGN Currency = "NGN"
	NIO Currency = "NIO"
	NOK Currency = "NOK"
	NPR Currency = "NPR"
	NZD Currency = "NZD"
	OMR Currency = "OMR"
	PEN Currency = "PEN"
	PGK Currency = "PGK"
	PHP Currency = "PHP"
	PKR Currency = "PKR"
	PLN Currency = "PLN"
	PYG Currency = "PYG"
	QAR Currency = "QAR"
	RON Currency = "RON"
	RSD Currency = "RSD"
	RUB Currency = "RUB"
	RWF Currency = "RWF"
	SAR Currency = "SAR"
	SCR Currency = "SCR"
	SEK Currency = "SEK"
	SGD Currency = "SGD"
	SLL Currency = "SLL"
	SOS Currency = "SOS"
	SSP Currency = "SSP"
	SVC Currency = "SVC"
	SZL Currency = "SZL"
	THB Currency = "THB"
	TND Currency = "TND"
	TRY Currency = "TRY"
	TTD Currency = "TTD"
	TWD Currency = "TWD"
	TZS Currency = "TZS"
	UAH Currency = "UAH"
	UGX Currency = "UGX"
	USD Currency = "USD"
	UYU Currency = "UYU"
	UZS Currency = "UZS"
	VND Currency = "VND"
	VUV Currency = "VUV"
	XAF Currency = "XAF"
	XCD Currency = "XCD"
	XOF Currency = "XOF"
	XPF Currency = "XPF"
	YER Currency = "YER"
	ZAR Currency = "ZAR"
	ZMW Currency = "ZMW"
)

var currencyExponents = map[Currency]int{
	BHD: 3, IQD: 3, JOD: 3, KWD: 3, OMR: 3, TND: 3,
	BIF: 0, CLP: 0, DJF: 0, GNF: 0, ISK: 0, KMF: 0,
	KRW: 0, PYG: 0, RWF: 0, UGX: 0, VND: 0, VUV: 0,
	XAF: 0, XOF: 0, XPF: 0,
	// All others default to 2
}

func (c Currency) Exponent() int {
	if exp, ok := currencyExponents[c]; ok {
		return exp
	}
	return 2
}

type RazorpayOrder struct {
	Id         string    `json:"id"`
	Entity     string    `json:"entity"`
	Amount     int       `json:"amount"`
	AmountPaid int       `json:"amount_paid"`
	AmountDue  int       `json:"amount_due"`
	Currency   string    `json:"currency"`
	Receipt    string    `json:"receipt"`
	OfferId    string    `json:"offer_id"`
	Status     string    `json:"status"`
	Attempts   int       `json:"attempts"`
	Notes      []string  `json:"notes"`
	CreatedAt  time.Time `json:"created_at"`
}

var client = razorpay.NewClient(config.GetEnvRaw("RAZORPAY_API_KEY"), config.GetEnvRaw("RAZORPAY_SECRET_KEY"))

func InitRazorpay(amt int, currency Currency, receipt, notes string) (*RazorpayOrder, error) {
	// Handling currency exponents
	exp := currency.Exponent()
	amt_string := strconv.Itoa(amt)
	for range exp {
		amt_string += "0"
	}
	amt_final, err := strconv.Atoi(amt_string)
	if err != nil {
		return nil, err
	}
	body, err := client.Order.Create(map[string]any{
		"amount":          amt_final,
		"currency":        string(currency),
		"receipt":         "some_receipt_id",
		"partial_payment": false,
		"notes": map[string]any{
			"key1": "value1",
			"key2": "value2",
		},
	}, nil)
	if err != nil {
		return nil, err
	}
	jsonData, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	var order RazorpayOrder
	err = json.Unmarshal(jsonData, &order)
	if err != nil {
		return nil, err
	}

	return &order, nil
}
