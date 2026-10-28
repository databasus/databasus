package httpapi

import (
	"databasus-backend/internal/modules/backups/internal/physical"
)

func toWalStreamStatistics(statistics physical.GetWalStreamStatisticsResponse) (WalStreamStatistics, error) {
	body := WalStreamStatistics{}

	if statistics.Current != nil {
		converted, err := toWalStreamCurrent(*statistics.Current)
		if err != nil {
			return WalStreamStatistics{}, err
		}

		body.Current = &converted
	}

	convertedHealth, err := toWalStreamHealth(statistics.Health)
	if err != nil {
		return WalStreamStatistics{}, err
	}

	body.Health = convertedHealth
	convertedKeepingUp, err := toWalStreamKeepingUp(statistics.KeepingUp)
	if err != nil {
		return WalStreamStatistics{}, err
	}

	body.KeepingUp = convertedKeepingUp
	body.LatestRestorableAt = statistics.LatestRestorableAt
	body.RestorableRanges = make([]WalStreamTimeRange, 0, len(statistics.RestorableRanges))

	for _, timeRange := range statistics.RestorableRanges {
		converted, err := toWalStreamTimeRange(timeRange)
		if err != nil {
			return WalStreamStatistics{}, err
		}

		body.RestorableRanges = append(body.RestorableRanges, converted)
	}

	convertedSeries, err := toWalStreamSeries(statistics.Series)
	if err != nil {
		return WalStreamStatistics{}, err
	}

	body.Series = convertedSeries
	convertedStreamer, err := toWalStreamStreamer(statistics.Streamer)
	if err != nil {
		return WalStreamStatistics{}, err
	}

	body.Streamer = convertedStreamer

	return body, nil
}
