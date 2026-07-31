package dns

import (
	"sort"

	"dns-failover/internal/model"
)

// sortRecords возвращает копию списка,
// отсортированную по priority
func sortRecords(records []model.Record) []model.Record {

	sorted := make(
		[]model.Record,
		len(records),
	)

	copy(
		sorted,
		records,
	)

	sort.Slice(
		sorted,
		func(i, j int) bool {
			return sorted[i].Priority < sorted[j].Priority
		},
	)

	return sorted
}


// PrimaryIP возвращает основной доступный IP
// (минимальный priority и disabled=false)
func PrimaryIP(
	records []model.Record,
) string {

	sorted := sortRecords(records)


	for _, record := range sorted {

		if !record.Disabled {

			return record.IP
		}
	}


	return ""
}


// NextIP возвращает следующий доступный IP
// после текущего согласно priority
func NextIP(
	records []model.Record,
	currentIP string,
) string {

	sorted := sortRecords(records)


	foundCurrent := false


	for _, record := range sorted {


		// нашли текущий активный IP
		if record.IP == currentIP {

			foundCurrent = true
			continue
		}


		// ищем следующий после него
		if foundCurrent &&
			!record.Disabled {

			return record.IP
		}
	}


	// если текущий был последний
	// пробуем начать с первого доступного
	for _, record := range sorted {

		if !record.Disabled &&
			record.IP != currentIP {

			return record.IP
		}
	}


	// резервов нет
	return currentIP
}


// GetActiveIP возвращает IP,
// который сейчас должен быть активным
// согласно приоритетам
func GetActiveIP(
	records []model.Record,
	currentIP string,
) string {


	if currentIP == "" {

		return PrimaryIP(records)
	}


	for _, record := range records {

		if record.IP == currentIP &&
			!record.Disabled {

			return currentIP
		}
	}


	return PrimaryIP(records)
}