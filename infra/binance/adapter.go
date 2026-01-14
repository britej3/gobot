package binance

import (
	"context"

	"github.com/britebrt/cognee/services/screener"
)

type ScreenerAdapter struct {
	client *ScreenerClient
}

func NewScreenerAdapter(client *ScreenerClient) *ScreenerAdapter {
	return &ScreenerAdapter{client: client}
}

func (a *ScreenerAdapter) GetExchangeInfo(ctx context.Context) ([]screener.ExchangeInfo, error) {
	info, err := a.client.GetExchangeInfo(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]screener.ExchangeInfo, 0, len(info))
	for _, p := range info {
		result = append(result, screener.ExchangeInfo{
			Symbol:         p.Symbol,
			ContractType:   p.ContractType,
			QuoteAsset:     p.QuoteAsset,
			Status:         p.Status,
			Volume24h:      p.Volume24h,
			PriceChangePct: p.PriceChangePct,
			LastUpdated:    p.LastUpdated,
		})
	}

	return result, nil
}

func (a *ScreenerAdapter) GetUSDMFuturesPairs(ctx context.Context) ([]screener.ExchangeInfo, error) {
	return a.GetExchangeInfo(ctx)
}

func (a *ScreenerAdapter) GetTopMemeCoins(ctx context.Context, limit int) ([]screener.ExchangeInfo, error) {
	pairs, err := a.client.GetTopMemeCoins(ctx, limit)
	if err != nil {
		return nil, err
	}

	result := make([]screener.ExchangeInfo, 0, len(pairs))
	for _, p := range pairs {
		result = append(result, screener.ExchangeInfo{
			Symbol:         p.Symbol,
			ContractType:   p.ContractType,
			QuoteAsset:     p.QuoteAsset,
			Status:         p.Status,
			Volume24h:      p.Volume24h,
			PriceChangePct: p.PriceChangePct,
			LastUpdated:    p.LastUpdated,
		})
	}

	return result, nil
}
