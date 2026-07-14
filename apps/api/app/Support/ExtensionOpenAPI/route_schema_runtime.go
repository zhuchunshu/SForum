package extensionopenapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"
)

func validateRouteSchemaWithLimits(
	ctx context.Context,
	slots chan struct{},
	timeout time.Duration,
	validate func(context.Context) error,
) error {
	if ctx == nil || slots == nil || cap(slots) == 0 || timeout <= 0 || validate == nil {
		return ErrRouteSchemaCatalogInvalid
	}
	validationCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	select {
	case slots <- struct{}{}:
	case <-validationCtx.Done():
		return validationCtx.Err()
	}
	result := make(chan error, 1)
	go func() {
		defer func() { <-slots }()
		result <- validate(validationCtx)
	}()
	select {
	case err := <-result:
		return err
	case <-validationCtx.Done():
		return validationCtx.Err()
	}
}

func decodeRouteSchemaJSON(body []byte, limit int) (any, error) {
	return decodeRouteSchemaJSONContext(context.Background(), body, limit, maxRouteDocumentNodes, maxRouteDocumentItems)
}

func decodeRouteSchemaJSONContext(ctx context.Context, body []byte, byteLimit, nodeLimit, itemLimit int) (any, error) {
	if ctx == nil || len(body) == 0 || len(body) > byteLimit || nodeLimit <= 0 || itemLimit <= 0 {
		return nil, ErrResourceBudget
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	budget := routeJSONBudget{ctx: ctx, nodesRemaining: nodeLimit, itemsRemaining: itemLimit}
	value, err := decodeRouteSchemaJSONValue(decoder, &budget, 0)
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if _, err := decoder.Token(); err != io.EOF {
		return nil, fmt.Errorf("trailing JSON value")
	}
	return value, nil
}

type routeJSONBudget struct {
	ctx            context.Context
	nodesRemaining int
	itemsRemaining int
}

func (b *routeJSONBudget) consumeNode() error {
	if err := b.ctx.Err(); err != nil {
		return err
	}
	b.nodesRemaining--
	if b.nodesRemaining < 0 {
		return fmt.Errorf("%w: JSON node budget exceeded", ErrResourceBudget)
	}
	return nil
}

func (b *routeJSONBudget) consumeItem() error {
	if err := b.ctx.Err(); err != nil {
		return err
	}
	b.itemsRemaining--
	if b.itemsRemaining < 0 {
		return fmt.Errorf("%w: JSON item budget exceeded", ErrResourceBudget)
	}
	return nil
}

func decodeRouteSchemaJSONValue(decoder *json.Decoder, budget *routeJSONBudget, depth int) (any, error) {
	if depth > maxRoutePayloadDepth {
		return nil, fmt.Errorf("%w: JSON nesting exceeds %d", ErrResourceBudget, maxRoutePayloadDepth)
	}
	if err := budget.consumeNode(); err != nil {
		return nil, err
	}
	token, err := decoder.Token()
	if err != nil {
		return nil, err
	}
	delimiter, composite := token.(json.Delim)
	if !composite {
		return token, nil
	}
	switch delimiter {
	case '{':
		object := make(map[string]any)
		for decoder.More() {
			if err := budget.consumeItem(); err != nil {
				return nil, err
			}
			keyToken, err := decoder.Token()
			if err != nil {
				return nil, err
			}
			key, ok := keyToken.(string)
			if !ok {
				return nil, fmt.Errorf("JSON object key must be a string")
			}
			if _, duplicate := object[key]; duplicate {
				return nil, fmt.Errorf("duplicate JSON object key %q", key)
			}
			value, err := decodeRouteSchemaJSONValue(decoder, budget, depth+1)
			if err != nil {
				return nil, err
			}
			object[key] = value
		}
		closing, err := decoder.Token()
		if err != nil || closing != json.Delim('}') {
			return nil, fmt.Errorf("invalid JSON object")
		}
		return object, nil
	case '[':
		array := make([]any, 0)
		for decoder.More() {
			if err := budget.consumeItem(); err != nil {
				return nil, err
			}
			value, err := decodeRouteSchemaJSONValue(decoder, budget, depth+1)
			if err != nil {
				return nil, err
			}
			array = append(array, value)
		}
		closing, err := decoder.Token()
		if err != nil || closing != json.Delim(']') {
			return nil, fmt.Errorf("invalid JSON array")
		}
		return array, nil
	default:
		return nil, fmt.Errorf("invalid JSON delimiter %q", delimiter)
	}
}
