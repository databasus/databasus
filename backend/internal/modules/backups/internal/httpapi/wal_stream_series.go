package httpapi

import (
	"databasus-backend/internal/modules/backups/internal/physical"
)

func toWalStreamSeries(series physical.WalStreamSeries) (WalStreamSeries, error) {
	body := WalStreamSeries{}
	body.BucketSeconds = series.BucketSeconds
	body.From = series.From

	switch string(series.Period) {
	case "HOUR", "DAY", "WEEK":
	default:
		return WalStreamSeries{}, errUnknownPhysicalValue
	}

	body.Period = WalStreamStatisticsPeriod(series.Period)
	body.Points = make([]WalStreamSeriesPoint, 0, len(series.Points))

	for _, point := range series.Points {
		converted, err := toWalStreamSeriesPoint(*point)
		if err != nil {
			return WalStreamSeries{}, err
		}

		body.Points = append(body.Points, converted)
	}

	body.To = series.To

	return body, nil
}
