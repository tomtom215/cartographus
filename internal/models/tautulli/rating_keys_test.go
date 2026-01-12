// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package tautulli

import (
	"encoding/json"
	"testing"
)

func TestTautulliNewRatingKeys_JSONUnmarshal(t *testing.T) {
	jsonData := `{
		"response": {
			"result": "success",
			"data": {
				"rating_keys": [
					{
						"old_rating_key": "12345",
						"new_rating_key": "67890",
						"title": "Inception",
						"media_type": "movie",
						"updated_at": 1700000000
					},
					{
						"old_rating_key": "11111",
						"new_rating_key": "22222",
						"title": "Breaking Bad",
						"media_type": "show",
						"updated_at": 1700100000
					}
				]
			}
		}
	}`

	var keys TautulliNewRatingKeys
	err := json.Unmarshal([]byte(jsonData), &keys)
	if err != nil {
		t.Fatalf("Failed to unmarshal TautulliNewRatingKeys: %v", err)
	}

	if keys.Response.Result != "success" {
		t.Errorf("Expected Result 'success', got '%s'", keys.Response.Result)
	}

	if len(keys.Response.Data.RatingKeys) != 2 {
		t.Fatalf("Expected 2 rating key mappings, got %d", len(keys.Response.Data.RatingKeys))
	}

	mapping := keys.Response.Data.RatingKeys[0]
	if mapping.OldRatingKey != "12345" {
		t.Errorf("Expected OldRatingKey '12345', got '%s'", mapping.OldRatingKey)
	}
	if mapping.NewRatingKey != "67890" {
		t.Errorf("Expected NewRatingKey '67890', got '%s'", mapping.NewRatingKey)
	}
	if mapping.Title != "Inception" {
		t.Errorf("Expected Title 'Inception', got '%s'", mapping.Title)
	}
	if mapping.MediaType != "movie" {
		t.Errorf("Expected MediaType 'movie', got '%s'", mapping.MediaType)
	}
	if mapping.UpdatedAt != 1700000000 {
		t.Errorf("Expected UpdatedAt 1700000000, got %d", mapping.UpdatedAt)
	}
}

func TestTautulliNewRatingKeys_EmptyData(t *testing.T) {
	jsonData := `{
		"response": {
			"result": "success",
			"data": {
				"rating_keys": []
			}
		}
	}`

	var keys TautulliNewRatingKeys
	err := json.Unmarshal([]byte(jsonData), &keys)
	if err != nil {
		t.Fatalf("Failed to unmarshal TautulliNewRatingKeys: %v", err)
	}

	if len(keys.Response.Data.RatingKeys) != 0 {
		t.Errorf("Expected 0 rating key mappings, got %d", len(keys.Response.Data.RatingKeys))
	}
}

func TestTautulliNewRatingKeys_ErrorResponse(t *testing.T) {
	jsonData := `{
		"response": {
			"result": "error",
			"message": "Rating key not found",
			"data": {
				"rating_keys": []
			}
		}
	}`

	var keys TautulliNewRatingKeys
	err := json.Unmarshal([]byte(jsonData), &keys)
	if err != nil {
		t.Fatalf("Failed to unmarshal TautulliNewRatingKeys: %v", err)
	}

	if keys.Response.Result != "error" {
		t.Errorf("Expected Result 'error', got '%s'", keys.Response.Result)
	}
	if keys.Response.Message == nil || *keys.Response.Message != "Rating key not found" {
		t.Error("Expected Message 'Rating key not found'")
	}
}

func TestTautulliNewRatingKeys_JSONRoundTrip(t *testing.T) {
	original := TautulliNewRatingKeys{
		Response: TautulliNewRatingKeysResponse{
			Result: "success",
			Data: TautulliNewRatingKeysData{
				RatingKeys: []TautulliRatingKeyMapping{
					{
						OldRatingKey: "old_001",
						NewRatingKey: "new_001",
						Title:        "Test Movie",
						MediaType:    "movie",
						UpdatedAt:    1700000000,
					},
				},
			},
		},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal TautulliNewRatingKeys: %v", err)
	}

	var decoded TautulliNewRatingKeys
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal TautulliNewRatingKeys: %v", err)
	}

	if len(decoded.Response.Data.RatingKeys) != len(original.Response.Data.RatingKeys) {
		t.Errorf("RatingKeys length mismatch")
	}
	if decoded.Response.Data.RatingKeys[0].OldRatingKey != original.Response.Data.RatingKeys[0].OldRatingKey {
		t.Errorf("OldRatingKey mismatch")
	}
}

func TestTautulliOldRatingKeys_JSONUnmarshal(t *testing.T) {
	jsonData := `{
		"response": {
			"result": "success",
			"data": {
				"rating_keys": [
					{
						"old_rating_key": "99999",
						"new_rating_key": "88888",
						"title": "The Matrix",
						"media_type": "movie",
						"updated_at": 1699000000
					}
				]
			}
		}
	}`

	var keys TautulliOldRatingKeys
	err := json.Unmarshal([]byte(jsonData), &keys)
	if err != nil {
		t.Fatalf("Failed to unmarshal TautulliOldRatingKeys: %v", err)
	}

	if keys.Response.Result != "success" {
		t.Errorf("Expected Result 'success', got '%s'", keys.Response.Result)
	}

	if len(keys.Response.Data.RatingKeys) != 1 {
		t.Fatalf("Expected 1 rating key mapping, got %d", len(keys.Response.Data.RatingKeys))
	}

	mapping := keys.Response.Data.RatingKeys[0]
	if mapping.OldRatingKey != "99999" {
		t.Errorf("Expected OldRatingKey '99999', got '%s'", mapping.OldRatingKey)
	}
	if mapping.NewRatingKey != "88888" {
		t.Errorf("Expected NewRatingKey '88888', got '%s'", mapping.NewRatingKey)
	}
	if mapping.Title != "The Matrix" {
		t.Errorf("Expected Title 'The Matrix', got '%s'", mapping.Title)
	}
}

func TestTautulliOldRatingKeys_EmptyData(t *testing.T) {
	jsonData := `{
		"response": {
			"result": "success",
			"data": {
				"rating_keys": []
			}
		}
	}`

	var keys TautulliOldRatingKeys
	err := json.Unmarshal([]byte(jsonData), &keys)
	if err != nil {
		t.Fatalf("Failed to unmarshal TautulliOldRatingKeys: %v", err)
	}

	if len(keys.Response.Data.RatingKeys) != 0 {
		t.Errorf("Expected 0 rating key mappings, got %d", len(keys.Response.Data.RatingKeys))
	}
}

func TestTautulliOldRatingKeys_ErrorResponse(t *testing.T) {
	jsonData := `{
		"response": {
			"result": "error",
			"message": "No old rating keys found",
			"data": {
				"rating_keys": []
			}
		}
	}`

	var keys TautulliOldRatingKeys
	err := json.Unmarshal([]byte(jsonData), &keys)
	if err != nil {
		t.Fatalf("Failed to unmarshal TautulliOldRatingKeys: %v", err)
	}

	if keys.Response.Result != "error" {
		t.Errorf("Expected Result 'error', got '%s'", keys.Response.Result)
	}
	if keys.Response.Message == nil || *keys.Response.Message != "No old rating keys found" {
		t.Error("Expected Message 'No old rating keys found'")
	}
}

func TestTautulliOldRatingKeys_JSONRoundTrip(t *testing.T) {
	original := TautulliOldRatingKeys{
		Response: TautulliOldRatingKeysResponse{
			Result: "success",
			Data: TautulliOldRatingKeysData{
				RatingKeys: []TautulliRatingKeyMapping{
					{
						OldRatingKey: "old_key",
						NewRatingKey: "new_key",
						Title:        "Test Show",
						MediaType:    "show",
						UpdatedAt:    1698000000,
					},
				},
			},
		},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal TautulliOldRatingKeys: %v", err)
	}

	var decoded TautulliOldRatingKeys
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal TautulliOldRatingKeys: %v", err)
	}

	if len(decoded.Response.Data.RatingKeys) != len(original.Response.Data.RatingKeys) {
		t.Errorf("RatingKeys length mismatch")
	}
	if decoded.Response.Data.RatingKeys[0].NewRatingKey != original.Response.Data.RatingKeys[0].NewRatingKey {
		t.Errorf("NewRatingKey mismatch")
	}
}

func TestTautulliRatingKeyMapping_MediaTypes(t *testing.T) {
	mediaTypes := []string{"movie", "show", "season", "episode", "artist", "album", "track"}

	for _, mType := range mediaTypes {
		t.Run("MediaType_"+mType, func(t *testing.T) {
			original := TautulliNewRatingKeys{
				Response: TautulliNewRatingKeysResponse{
					Result: "success",
					Data: TautulliNewRatingKeysData{
						RatingKeys: []TautulliRatingKeyMapping{
							{
								OldRatingKey: "old",
								NewRatingKey: "new",
								Title:        "Test",
								MediaType:    mType,
								UpdatedAt:    1700000000,
							},
						},
					},
				},
			}

			data, err := json.Marshal(original)
			if err != nil {
				t.Fatalf("Failed to marshal: %v", err)
			}

			var decoded TautulliNewRatingKeys
			err = json.Unmarshal(data, &decoded)
			if err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}

			if decoded.Response.Data.RatingKeys[0].MediaType != mType {
				t.Errorf("Expected MediaType '%s', got '%s'", mType, decoded.Response.Data.RatingKeys[0].MediaType)
			}
		})
	}
}

func TestTautulliRatingKeyMapping_MultipleUpdates(t *testing.T) {
	jsonData := `{
		"response": {
			"result": "success",
			"data": {
				"rating_keys": [
					{
						"old_rating_key": "1",
						"new_rating_key": "2",
						"title": "Movie A",
						"media_type": "movie",
						"updated_at": 1700000000
					},
					{
						"old_rating_key": "2",
						"new_rating_key": "3",
						"title": "Movie A",
						"media_type": "movie",
						"updated_at": 1700100000
					},
					{
						"old_rating_key": "3",
						"new_rating_key": "4",
						"title": "Movie A",
						"media_type": "movie",
						"updated_at": 1700200000
					}
				]
			}
		}
	}`

	var keys TautulliNewRatingKeys
	err := json.Unmarshal([]byte(jsonData), &keys)
	if err != nil {
		t.Fatalf("Failed to unmarshal TautulliNewRatingKeys: %v", err)
	}

	if len(keys.Response.Data.RatingKeys) != 3 {
		t.Fatalf("Expected 3 rating key mappings, got %d", len(keys.Response.Data.RatingKeys))
	}

	// Verify chain: 1 -> 2 -> 3 -> 4
	for i, mapping := range keys.Response.Data.RatingKeys {
		expectedOld := string(rune('1' + i))
		expectedNew := string(rune('2' + i))
		if mapping.OldRatingKey != expectedOld {
			t.Errorf("Expected OldRatingKey '%s', got '%s'", expectedOld, mapping.OldRatingKey)
		}
		if mapping.NewRatingKey != expectedNew {
			t.Errorf("Expected NewRatingKey '%s', got '%s'", expectedNew, mapping.NewRatingKey)
		}
	}
}

func TestTautulliRatingKeyMapping_SpecialCharactersInTitle(t *testing.T) {
	jsonData := `{
		"response": {
			"result": "success",
			"data": {
				"rating_keys": [
					{
						"old_rating_key": "111",
						"new_rating_key": "222",
						"title": "Movie: The Sequel (2024) - Director's Cut",
						"media_type": "movie",
						"updated_at": 1700000000
					},
					{
						"old_rating_key": "333",
						"new_rating_key": "444",
						"title": "Artist's \"Best\" Album",
						"media_type": "album",
						"updated_at": 1700000000
					}
				]
			}
		}
	}`

	var keys TautulliNewRatingKeys
	err := json.Unmarshal([]byte(jsonData), &keys)
	if err != nil {
		t.Fatalf("Failed to unmarshal TautulliNewRatingKeys: %v", err)
	}

	if keys.Response.Data.RatingKeys[0].Title != "Movie: The Sequel (2024) - Director's Cut" {
		t.Errorf("Title mismatch, got '%s'", keys.Response.Data.RatingKeys[0].Title)
	}
	if keys.Response.Data.RatingKeys[1].Title != "Artist's \"Best\" Album" {
		t.Errorf("Title mismatch, got '%s'", keys.Response.Data.RatingKeys[1].Title)
	}
}

func TestTautulliRatingKeyMapping_LargeRatingKeys(t *testing.T) {
	jsonData := `{
		"response": {
			"result": "success",
			"data": {
				"rating_keys": [
					{
						"old_rating_key": "9999999999",
						"new_rating_key": "10000000000",
						"title": "Large Key Movie",
						"media_type": "movie",
						"updated_at": 1700000000
					}
				]
			}
		}
	}`

	var keys TautulliNewRatingKeys
	err := json.Unmarshal([]byte(jsonData), &keys)
	if err != nil {
		t.Fatalf("Failed to unmarshal TautulliNewRatingKeys: %v", err)
	}

	mapping := keys.Response.Data.RatingKeys[0]
	if mapping.OldRatingKey != "9999999999" {
		t.Errorf("Expected OldRatingKey '9999999999', got '%s'", mapping.OldRatingKey)
	}
	if mapping.NewRatingKey != "10000000000" {
		t.Errorf("Expected NewRatingKey '10000000000', got '%s'", mapping.NewRatingKey)
	}
}

func TestTautulliNewRatingKeys_NullMessage(t *testing.T) {
	jsonData := `{
		"response": {
			"result": "success",
			"message": null,
			"data": {
				"rating_keys": []
			}
		}
	}`

	var keys TautulliNewRatingKeys
	err := json.Unmarshal([]byte(jsonData), &keys)
	if err != nil {
		t.Fatalf("Failed to unmarshal TautulliNewRatingKeys: %v", err)
	}

	if keys.Response.Message != nil {
		t.Errorf("Expected Message nil, got '%s'", *keys.Response.Message)
	}
}

func TestTautulliOldRatingKeys_NullMessage(t *testing.T) {
	jsonData := `{
		"response": {
			"result": "success",
			"message": null,
			"data": {
				"rating_keys": []
			}
		}
	}`

	var keys TautulliOldRatingKeys
	err := json.Unmarshal([]byte(jsonData), &keys)
	if err != nil {
		t.Fatalf("Failed to unmarshal TautulliOldRatingKeys: %v", err)
	}

	if keys.Response.Message != nil {
		t.Errorf("Expected Message nil, got '%s'", *keys.Response.Message)
	}
}
