package market

import (
	"time"
)

type Kline struct {
	OpenTime  time.Time
	Open      float64
	High      float64
	Low       float64
	Close     float64
	Volume    float64
	CloseTime time.Time
}

type Market struct {
	Symbol    string
	Klines    []Kline
	UpdatedAt time.Time
}

func (m *Market) LastKline() *Kline {
	if len(m.Klines) == 0 {
		return nil
	}
	return &m.Klines[len(m.Klines)-1]
}

func (m *Market) PreviousKline(n int) *Kline {
	if len(m.Klines) <= n {
		return nil
	}
	return &m.Klines[len(m.Klines)-1-n]
}

func (m *Market) Volume24h() float64 {
	var volume float64
	for _, k := range m.Klines {
		volume += k.Volume
	}
	return volume
}

func (m *Market) PriceChange() float64 {
	if len(m.Klines) < 2 {
		return 0
	}
	first := m.Klines[0].Close
	last := m.LastKline().Close
	return ((last - first) / first) * 100
}

func (m *Market) Highest(period int) float64 {
	if len(m.Klines) < period {
		period = len(m.Klines)
	}
	highest := m.Klines[0].High
	for i := 1; i < period; i++ {
		if m.Klines[i].High > highest {
			highest = m.Klines[i].High
		}
	}
	return highest
}

func (m *Market) Lowest(period int) float64 {
	if len(m.Klines) < period {
		period = len(m.Klines)
	}
	lowest := m.Klines[0].Low
	for i := 1; i < period; i++ {
		if m.Klines[i].Low < lowest {
			lowest = m.Klines[i].Low
		}
	}
	return lowest
}

func (m *Market) RSI(period int) float64 {
	if len(m.Klines) < period+1 {
		return 50
	}

	var gains, losses float64
	for i := len(m.Klines) - period; i < len(m.Klines); i++ {
		change := m.Klines[i].Close - m.Klines[i-1].Close
		if change > 0 {
			gains += change
		} else {
			losses -= change
		}
	}

	avgGain := gains / float64(period)
	avgLoss := losses / float64(period)

	if avgLoss == 0 {
		return 100
	}

	rs := avgGain / avgLoss
	return 100 - (100 / (1 + rs))
}

func (m *Market) EMA(period int) float64 {
	if len(m.Klines) < period {
		return m.LastKline().Close
	}

	k := 2.0 / float64(period+1)
	ema := m.Klines[period-1].Close

	for i := period; i < len(m.Klines); i++ {
		ema = m.Klines[i].Close*k + ema*(1-k)
	}

	return ema
}

func (m *Market) MACD() (macd, signal, histogram float64) {
	ema12 := m.EMA(12)
	ema26 := m.EMA(26)
	macd = ema12 - ema26

	signalEMA := m.EMA(9)
	signal = signalEMA

	histogram = macd - signal
	return
}

func (m *Market) Volatility() float64 {
	if len(m.Klines) < 2 {
		return 0
	}

	var returns []float64
	for i := 1; i < len(m.Klines); i++ {
		ret := (m.Klines[i].Close - m.Klines[i-1].Close) / m.Klines[i-1].Close
		returns = append(returns, ret)
	}

	mean := 0.0
	for _, r := range returns {
		mean += r
	}
	mean /= float64(len(returns))

	variance := 0.0
	for _, r := range returns {
		variance += (r - mean) * (r - mean)
	}
	variance /= float64(len(returns))

	return variance * 100
}

func (m *Market) ATR(period int) float64 {
	if len(m.Klines) < period+1 {
		return 0
	}

	var trueRanges []float64
	for i := 1; i < len(m.Klines); i++ {
		tr := m.Klines[i].High - m.Klines[i].Low
		highLow := m.Klines[i].High - m.Klines[i-1].Close
		if highLow < 0 {
			highLow = -highLow
		}
		lowClose := m.Klines[i-1].Close - m.Klines[i].Low
		if lowClose < 0 {
			lowClose = -lowClose
		}

		trueRange := tr
		if highLow > trueRange {
			trueRange = highLow
		}
		if lowClose > trueRange {
			trueRange = lowClose
		}
		trueRanges = append(trueRanges, trueRange)
	}

	var atr float64
	for _, tr := range trueRanges[len(trueRanges)-period:] {
		atr += tr
	}
	atr /= float64(period)

	return atr
}
