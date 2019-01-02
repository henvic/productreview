package reviews

import (
	"testing"
)

type validateCase struct {
	in       Review
	expected error
}

var validateCases = []validateCase{
	validateCase{
		Review{
			ProductID: 1,
			Name:      "foo",
			Email:     "foo@example.com",
			Rating:    0,
		},
		nil,
	},
	validateCase{
		Review{
			ProductID: 1,
			Name:      "foo",
			Email:     "foo@example.com",
			Rating:    5,
		},
		nil,
	},
	validateCase{
		Review{
			ProductID: -1,
			Name:      "foo",
			Email:     "foo@example.com",
			Rating:    3,
		},
		ValidationError{"invalid product ID"},
	},
	validateCase{
		Review{
			ProductID: 2,
			Name:      "",
			Email:     "foo@example.com",
			Rating:    3,
		},
		ValidationError{"name is empty"},
	},
	validateCase{
		Review{
			ProductID: 2,
			Name:      "Foo",
			Email:     "example.com",
			Rating:    3,
		},
		ValidationError{"invalid email address"},
	},
	validateCase{
		Review{
			ProductID: 2,
			Name:      "Foo",
			Email:     "foo@example.com",
			Rating:    -3,
		},
		ValidationError{"invalid rating value"},
	},
	validateCase{
		Review{
			ProductID: 2,
			Name:      "Foo",
			Email:     "foo@example.com",
			Rating:    6,
		},
		ValidationError{"invalid rating value"},
	},
}

func TestValidate(t *testing.T) {
	for _, v := range validateCases {
		if got := Validate(v.in); got != v.expected {
			t.Errorf("expected %v, got %v instead", v.expected, got)
		}
	}
}
