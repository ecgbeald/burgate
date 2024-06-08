package stripe

import (
	pb "github.com/ecgbeald/burgate/proto"
	"github.com/stripe/stripe-go/v78"
	"github.com/stripe/stripe-go/v78/paymentlink"
	"github.com/stripe/stripe-go/v78/price"
	"github.com/stripe/stripe-go/v78/product"
)

func CreateStripeProduct(name string, item_price int64) (string, error) {
	stripe.Key = "VERY SECRET KEY"
	params := &stripe.ProductParams{Name: stripe.String(name)}
	product, err := product.New(params)
	if err != nil {
		return "", err
	}
	price_params := &stripe.PriceParams{Currency: stripe.String(string(stripe.CurrencyGBP)), Product: stripe.String(product.ID), UnitAmount: stripe.Int64(item_price)}
	product_price, err := price.New(price_params)
	if err != nil {
		return "", err
	}
	return product_price.ID, nil
}

func CreatePaymentLink(items []*pb.Item) (string, error) {
	stripe.Key = "VERY SECRET KEY"
	line_items := []*stripe.PaymentLinkLineItemParams{}
	for _, item := range items {
		line_items = append(line_items, &stripe.PaymentLinkLineItemParams{
			Price:    stripe.String(item.PriceID),
			Quantity: stripe.Int64(int64(item.Quantity)),
		})
	}
	result, err := paymentlink.New(&stripe.PaymentLinkParams{
		LineItems: line_items,
	})
	if err != nil {
		return "", err
	}
	return string(result.URL), nil
}
