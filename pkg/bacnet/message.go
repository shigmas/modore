package bacnet

type (
	// Filterable is a super simple test that filters on various BACnet message types. These are all
	// declared as types, but are uint8's. We manually cast to uint8 so we can use the types as
	// masks
	Filterable interface {
		MatchesFilter(filter uint8) bool
	}

	Message interface {
	}
)
