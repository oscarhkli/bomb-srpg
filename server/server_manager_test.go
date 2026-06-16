package server

import (
	"strings"
	"testing"
)

func TestServerStateManager_CreateMatchRoom(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		s := NewServerStateManager()

		id, err := s.CreateMatchRoom()
		if err != nil {
			t.Fatalf("CreateMatchRoom() returned error: %v", err)
		}

		if id == "" {
			t.Fatal("Expected non-empty room ID")
		}
		if len(id) != 5 {
			t.Errorf("Expected ID length 5, got %d: %s", len(id), id)
		}
		if !isValidCrockfordCode(id) {
			t.Errorf("ID contains invalid characters: %s", id)
		}

		room, ok := s.Rooms[id]
		if !ok {
			t.Fatal("Room not added to manager.Rooms")
		}
		if room.Match != nil {
			t.Errorf("Expected Match=nil, got %v", room.Match)
		}
		if room.ID != id {
			t.Errorf("Room.ID mismatch: got %s want %s", room.ID, id)
		}
	})

	t.Run("Room ID collision retry", func(t *testing.T) {
		s := NewServerStateManager()

		existingID := "ABCDE"
		s.Rooms[existingID] = &MatchRoom{ID: existingID}

		id, err := s.CreateMatchRoom()
		if err != nil {
			t.Fatalf("CreateMatchRoom returned error: %v", err)
		}

		if id == existingID {
			t.Fatal("Returned ID should not match pre-seeded ID")
		}
		if len(id) != 5 || !isValidCrockfordCode(id) {
			t.Errorf("Invalid generated ID: %s", id)
		}

		if _, ok := s.Rooms[existingID]; !ok {
			t.Error("Predefined room missing")
		}
		if _, ok := s.Rooms[id]; !ok {
			t.Error("New room not added")
		}
	})

	t.Run("Max retries exhausted", func(t *testing.T) {
		s := NewServerStateManager()

		// Override generator to return IDs in sequence
		callCount := 0
		roomIDs := []string{"ID001", "ID002", "ID003", "ID004", "ID005"}
		for _, id := range roomIDs {
			s.Rooms[id] = &MatchRoom{ID: id}
		}
		s.generateRoomID = func(int) string {
			if callCount < len(roomIDs) {
				id := roomIDs[callCount]
				callCount++
				return id
			}
			return "SHOULD_NOT_REACH"
		}

		id, err := s.CreateMatchRoom()
		if err == nil {
			t.Fatalf("Expected error after max retries, got ID: %s", id)
		}
		if id != "" {
			t.Errorf("Expected empty ID on error, got: %s", id)
		}

		for _, existing := range roomIDs {
			if _, ok := s.Rooms[existing]; !ok {
				t.Errorf("Predefined room %s missing", existing)
			}
		}
	})
}

func isValidCrockfordCode(s string) bool {
	const alphabet = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"
	for _, c := range s {
		if !strings.ContainsRune(alphabet, c) {
			return false
		}
	}
	return true
}
