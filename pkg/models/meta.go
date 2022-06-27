package models

import "reflect"

type Meta map[string]any

func (m Meta) Omit() {
	for k, v := range m {
		if reflect.ValueOf(v).IsZero() {
			delete(m, k)
		}

		switch x := v.(type) {
		case Meta:
			x.Omit()
		}
	}
}
