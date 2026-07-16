package queryregistry

import (
	"encoding"
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"unicode/utf8"
)

const (
	maximumResultJSONDepth = 64
	maximumResultJSONNodes = 1 << 20
	// One filter boundary needs at most three complete JSON traversals: detached
	// input, detached output, and the Host schema-validator copy. The allowance is
	// shared by the whole chain, so adding filters cannot reset any traversal.
	resultFilterBudgetDocuments = 3
)

type resultJSONBudget struct {
	remaining int
	nodes     int
	maxNodes  int
}

type resultJSONMeasure struct {
	bytes int
	nodes int
}

// preflightRowsBounded conservatively counts the canonical JSON tree before
// encoding/json allocates a second complete result buffer. Custom marshalers
// are rejected because their output cannot be bounded from the value graph.
func preflightRowsBounded(rows []QueryRow, maximumBytes int) error {
	_, err := measureRowsBounded(rows, maximumBytes)
	return err
}

func measureRowsBounded(rows []QueryRow, maximumBytes int) (resultJSONMeasure, error) {
	if maximumBytes < 1 {
		return resultJSONMeasure{}, ErrResultTooLarge
	}
	budget := &resultJSONBudget{
		remaining: maximumBytes,
		maxNodes:  resultJSONNodeLimit(maximumBytes),
	}
	if err := budget.walk(reflect.ValueOf(rows), 0); err != nil {
		return resultJSONMeasure{}, err
	}
	return resultJSONMeasure{bytes: maximumBytes - budget.remaining, nodes: budget.nodes}, nil
}

func newResultFilterJSONBudget(maximumBytes int) (*resultJSONBudget, error) {
	maximumInt := int(^uint(0) >> 1)
	if maximumBytes < 1 || maximumBytes > maximumInt/resultFilterBudgetDocuments {
		return nil, ErrExecutionInvalid
	}
	maxNodes := resultJSONNodeLimit(maximumBytes)
	if maxNodes > maximumInt/resultFilterBudgetDocuments {
		return nil, ErrExecutionInvalid
	}
	return &resultJSONBudget{
		remaining: maximumBytes * resultFilterBudgetDocuments,
		maxNodes:  maxNodes * resultFilterBudgetDocuments,
	}, nil
}

func resultJSONNodeLimit(maximumBytes int) int {
	if maximumBytes < maximumResultJSONNodes {
		return maximumBytes
	}
	return maximumResultJSONNodes
}

func (b *resultJSONBudget) consumeMeasure(measure resultJSONMeasure) error {
	if b == nil || measure.bytes < 0 || measure.nodes < 0 || b.maxNodes < 1 ||
		measure.nodes > b.maxNodes-b.nodes {
		return fmt.Errorf("%w: cumulative result-filter node count exceeds Host bounds", ErrResultTooLarge)
	}
	if err := b.consume(measure.bytes); err != nil {
		return fmt.Errorf("%w: cumulative result-filter bytes exceed Host bounds", ErrResultTooLarge)
	}
	b.nodes += measure.nodes
	return nil
}

func (b *resultJSONBudget) walk(value reflect.Value, depth int) error {
	if depth > maximumResultJSONDepth {
		return fmt.Errorf("%w: result nesting exceeds Host bounds", ErrResultTooLarge)
	}
	b.nodes++
	if b.nodes > b.maxNodes {
		return fmt.Errorf("%w: result node count exceeds Host bounds", ErrResultTooLarge)
	}
	if !value.IsValid() {
		return b.consume(4)
	}
	if value.Kind() == reflect.Interface {
		if value.IsNil() {
			return b.consume(4)
		}
		return b.walk(value.Elem(), depth+1)
	}
	if customResultMarshaler(value) {
		return fmt.Errorf("%w: custom marshalers are not allowed in query results", ErrResultInvalid)
	}

	switch value.Kind() {
	case reflect.Pointer:
		if value.IsNil() {
			return b.consume(4)
		}
		return b.walk(value.Elem(), depth+1)
	case reflect.Map:
		if value.IsNil() {
			return b.consume(4)
		}
		if value.Type().Key().Kind() != reflect.String {
			return fmt.Errorf("%w: result objects require string keys", ErrResultInvalid)
		}
		if err := b.consume(2); err != nil {
			return err
		}
		iterator := value.MapRange()
		index := 0
		for iterator.Next() {
			if index > 0 {
				if err := b.consume(1); err != nil {
					return err
				}
			}
			key := iterator.Key()
			if key.CanInterface() {
				if _, custom := key.Interface().(encoding.TextMarshaler); custom {
					return fmt.Errorf("%w: custom object-key marshalers are not allowed", ErrResultInvalid)
				}
			}
			if err := b.consumeJSONString(key.String()); err != nil {
				return err
			}
			if err := b.consume(1); err != nil {
				return err
			}
			if err := b.walk(iterator.Value(), depth+1); err != nil {
				return err
			}
			index++
		}
		return nil
	case reflect.Slice:
		if value.IsNil() {
			return b.consume(4)
		}
		if value.Type().Elem().Kind() == reflect.Uint8 {
			length := value.Len()
			maximumInt := int(^uint(0) >> 1)
			if length > maximumInt-2 || b.remaining < 2 {
				return ErrResultTooLarge
			}
			groups := (length + 2) / 3
			if groups > (b.remaining-2)/4 {
				return ErrResultTooLarge
			}
			return b.consume(2 + 4*groups)
		}
		fallthrough
	case reflect.Array:
		if err := b.consume(2); err != nil {
			return err
		}
		for index := 0; index < value.Len(); index++ {
			if index > 0 {
				if err := b.consume(1); err != nil {
					return err
				}
			}
			if err := b.walk(value.Index(index), depth+1); err != nil {
				return err
			}
		}
		return nil
	case reflect.String:
		return b.consumeJSONString(value.String())
	case reflect.Bool:
		return b.consume(5)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return b.consume(24)
	case reflect.Float32, reflect.Float64:
		if number := value.Float(); math.IsNaN(number) || math.IsInf(number, 0) {
			return fmt.Errorf("%w: non-finite result number", ErrResultInvalid)
		}
		return b.consume(32)
	default:
		return fmt.Errorf("%w: unsupported result value %s", ErrResultInvalid, value.Kind())
	}
}

func customResultMarshaler(value reflect.Value) bool {
	implements := func(candidate reflect.Value) bool {
		if !candidate.IsValid() || !candidate.CanInterface() {
			return false
		}
		item := candidate.Interface()
		_, jsonCustom := item.(json.Marshaler)
		_, textCustom := item.(encoding.TextMarshaler)
		return jsonCustom || textCustom
	}
	if implements(value) {
		return true
	}
	// encoding/json uses the pointer method set for addressable values, notably
	// slice and pointed-array elements. Detect it before Marshal can allocate an
	// unbounded replacement document outside the Host byte budget.
	return value.CanAddr() && implements(value.Addr())
}

func (b *resultJSONBudget) consumeJSONString(value string) error {
	if err := b.consume(2); err != nil {
		return err
	}
	for len(value) > 0 {
		current := value[0]
		switch {
		case current < 0x20 || current == '<' || current == '>' || current == '&':
			if err := b.consume(6); err != nil {
				return err
			}
			value = value[1:]
		case current == '\\' || current == '"':
			if err := b.consume(2); err != nil {
				return err
			}
			value = value[1:]
		case current < utf8.RuneSelf:
			if err := b.consume(1); err != nil {
				return err
			}
			value = value[1:]
		default:
			runeValue, size := utf8.DecodeRuneInString(value)
			if runeValue == utf8.RuneError && size == 1 || runeValue == '\u2028' || runeValue == '\u2029' {
				if err := b.consume(6); err != nil {
					return err
				}
			} else if err := b.consume(size); err != nil {
				return err
			}
			value = value[size:]
		}
	}
	return nil
}

func (b *resultJSONBudget) consume(size int) error {
	if size < 0 || size > b.remaining {
		return ErrResultTooLarge
	}
	b.remaining -= size
	return nil
}
