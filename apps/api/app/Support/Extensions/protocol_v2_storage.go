package extensionsruntime

import (
	"encoding/base64"
)

// attachment.storage.provider known-slot：经 ProviderCall 承载分块 Put/Open。
// 二进制块以 base64 放进 TypedDocument（structpb 不接受 []byte）。
const (
	storageProviderSlot           = "attachment.storage.provider"
	storageLegacyContractVersion  = "1"
	storageOpProbe                = "probe"
	storageOpPutBegin             = "put_begin"
	storageOpPutChunk             = "put_chunk"
	storageOpOpen                 = "open"
	storageOpGetChunk             = "get_chunk"
	storageOpClose                = "close"
	storageOpDelete               = "delete"
	storageOpStat                 = "stat"
	storageOpExists               = "exists"
	storageOpPublicURL            = "public_url"
	storageOpSignedURL            = "signed_url"
)

func (c *protocolV2Client) StorageProbe(StorageProbeRequest) (StorageProbeResponse, error) {
	response, err := c.providerCall(storageProviderSlot, storageOpProbe, map[string]any{})
	if err != nil {
		return StorageProbeResponse{}, err
	}
	values := protocolV2Values(response.GetOutput())
	return StorageProbeResponse{
		OK: booleanValue(values, "ok"), Reason: stringValue(values, "reason"),
		Message: stringValue(values, "message"),
	}, nil
}

func (c *protocolV2Client) StoragePutBegin(input StoragePutBeginRequest) (StorageSessionResponse, error) {
	response, err := c.providerCall(storageProviderSlot, storageOpPutBegin, map[string]any{
		"key": input.Key, "contentType": input.ContentType, "size": input.Size,
	})
	if err != nil {
		return StorageSessionResponse{}, err
	}
	return storageSessionFromValues(protocolV2Values(response.GetOutput())), nil
}

func (c *protocolV2Client) StoragePutChunk(input StoragePutChunkRequest) (StorageResult, error) {
	response, err := c.providerCall(storageProviderSlot, storageOpPutChunk, map[string]any{
		"sessionId": input.SessionID,
		"data":      base64.StdEncoding.EncodeToString(input.Data),
		"final":     input.Final,
	})
	if err != nil {
		return StorageResult{}, err
	}
	return storageResultFromValues(protocolV2Values(response.GetOutput())), nil
}

func (c *protocolV2Client) StorageOpen(input StorageOpenRequest) (StorageSessionResponse, error) {
	response, err := c.providerCall(storageProviderSlot, storageOpOpen, map[string]any{
		"key": input.Key,
	})
	if err != nil {
		return StorageSessionResponse{}, err
	}
	return storageSessionFromValues(protocolV2Values(response.GetOutput())), nil
}

func (c *protocolV2Client) StorageGetChunk(input StorageGetChunkRequest) (StorageGetChunkResponse, error) {
	response, err := c.providerCall(storageProviderSlot, storageOpGetChunk, map[string]any{
		"sessionId": input.SessionID, "maxBytes": int64(input.MaxBytes),
	})
	if err != nil {
		return StorageGetChunkResponse{}, err
	}
	values := protocolV2Values(response.GetOutput())
	data, decodeErr := base64.StdEncoding.DecodeString(stringValue(values, "data"))
	if decodeErr != nil && stringValue(values, "data") != "" {
		return StorageGetChunkResponse{
			Reason: "storage.v2.invalid_chunk_encoding", Message: decodeErr.Error(),
		}, nil
	}
	return StorageGetChunkResponse{
		OK: booleanValue(values, "ok"), Data: data, EOF: booleanValue(values, "eof"),
		Reason: stringValue(values, "reason"), Message: stringValue(values, "message"),
	}, nil
}

func (c *protocolV2Client) StorageClose(input StorageCloseRequest) (StorageResult, error) {
	response, err := c.providerCall(storageProviderSlot, storageOpClose, map[string]any{
		"sessionId": input.SessionID,
	})
	if err != nil {
		return StorageResult{}, err
	}
	return storageResultFromValues(protocolV2Values(response.GetOutput())), nil
}

func (c *protocolV2Client) StorageDelete(input StorageObjectRequest) (StorageResult, error) {
	response, err := c.providerCall(storageProviderSlot, storageOpDelete, map[string]any{
		"key": input.Key,
	})
	if err != nil {
		return StorageResult{}, err
	}
	return storageResultFromValues(protocolV2Values(response.GetOutput())), nil
}

func (c *protocolV2Client) StorageStat(input StorageStatRequest) (StorageStatResponse, error) {
	response, err := c.providerCall(storageProviderSlot, storageOpStat, map[string]any{
		"key": input.Key,
	})
	if err != nil {
		return StorageStatResponse{}, err
	}
	values := protocolV2Values(response.GetOutput())
	return StorageStatResponse{
		OK: booleanValue(values, "ok"), Exists: booleanValue(values, "exists"),
		Size: int64Value(values, "size"), ContentType: stringValue(values, "contentType"),
		ModifiedUnix: int64Value(values, "modifiedUnix"),
		Reason:       stringValue(values, "reason"), Message: stringValue(values, "message"),
	}, nil
}

func (c *protocolV2Client) StorageExists(input StorageExistsRequest) (StorageExistsResponse, error) {
	response, err := c.providerCall(storageProviderSlot, storageOpExists, map[string]any{
		"key": input.Key,
	})
	if err != nil {
		return StorageExistsResponse{}, err
	}
	values := protocolV2Values(response.GetOutput())
	return StorageExistsResponse{
		OK: booleanValue(values, "ok"), Exists: booleanValue(values, "exists"),
		Reason: stringValue(values, "reason"), Message: stringValue(values, "message"),
	}, nil
}

func (c *protocolV2Client) StoragePublicURL(input StoragePublicURLRequest) (StorageURLResponse, error) {
	response, err := c.providerCall(storageProviderSlot, storageOpPublicURL, map[string]any{
		"key": input.Key,
	})
	if err != nil {
		return StorageURLResponse{}, err
	}
	return storageURLFromValues(protocolV2Values(response.GetOutput())), nil
}

func (c *protocolV2Client) StorageSignedURL(input StorageSignedURLRequest) (StorageURLResponse, error) {
	response, err := c.providerCall(storageProviderSlot, storageOpSignedURL, map[string]any{
		"key": input.Key, "ttlSeconds": input.TTLSeconds,
	})
	if err != nil {
		return StorageURLResponse{}, err
	}
	return storageURLFromValues(protocolV2Values(response.GetOutput())), nil
}

func storageSessionFromValues(values map[string]any) StorageSessionResponse {
	return StorageSessionResponse{
		OK: booleanValue(values, "ok"), SessionID: stringValue(values, "sessionId"),
		Size: int64Value(values, "size"), ContentType: stringValue(values, "contentType"),
		Reason: stringValue(values, "reason"), Message: stringValue(values, "message"),
	}
}

func storageResultFromValues(values map[string]any) StorageResult {
	return StorageResult{
		OK: booleanValue(values, "ok"), Reason: stringValue(values, "reason"),
		Message: stringValue(values, "message"),
	}
}

func storageURLFromValues(values map[string]any) StorageURLResponse {
	return StorageURLResponse{
		OK: booleanValue(values, "ok"), URL: stringValue(values, "url"),
		Reason: stringValue(values, "reason"), Message: stringValue(values, "message"),
	}
}

// int64Value 兼容 structpb AsMap 的 float64 与整型。
func int64Value(values map[string]any, key string) int64 {
	if values == nil {
		return 0
	}
	switch raw := values[key].(type) {
	case int64:
		return raw
	case int:
		return int64(raw)
	case int32:
		return int64(raw)
	case float64:
		return int64(raw)
	case float32:
		return int64(raw)
	default:
		return 0
	}
}
