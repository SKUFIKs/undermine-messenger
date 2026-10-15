package storage

import (
	"fmt"
	"time"
)

// GroupEntry is a group with its owner
type GroupEntry struct {
	ID      [16]byte
	Name    string
	OwnerID [16]byte
}

// GroupMemberEntry is a group member with user info
type GroupMemberEntry struct {
	UserID    [16]byte
	Username  string
	PublicKey [32]byte
}

func (s *SQLiteStorage) CreateGroup(id [16]byte, name string, ownerID [16]byte) error {
	_, err := s.db.Exec(
		"INSERT INTO groups (id, name, owner_id, created_at) VALUES (?, ?, ?, ?)",
		id[:], name, ownerID[:], time.Now(),
	)
	return err
}

func (s *SQLiteStorage) GetGroupsByMember(userID [16]byte) ([]GroupEntry, error) {
	rows, err := s.db.Query(`
		SELECT g.id, g.name, g.owner_id
		FROM groups g
		JOIN group_members gm ON gm.group_id = g.id
		WHERE gm.user_id = ?
		ORDER BY g.created_at DESC`,
		userID[:],
	)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения списка групп: %w", err)
	}
	defer rows.Close()

	var result []GroupEntry
	for rows.Next() {
		var g GroupEntry
		var idBytes, ownerBytes []byte
		if err := rows.Scan(&idBytes, &g.Name, &ownerBytes); err != nil {
			return nil, fmt.Errorf("ошибка чтения группы: %w", err)
		}
		if len(idBytes) != 16 || len(ownerBytes) != 16 {
			continue
		}
		copy(g.ID[:], idBytes)
		copy(g.OwnerID[:], ownerBytes)
		result = append(result, g)
	}
	return result, rows.Err()
}

func (s *SQLiteStorage) AddGroupMember(groupID, userID [16]byte) error {
	_, err := s.db.Exec(
		"INSERT OR IGNORE INTO group_members (group_id, user_id, joined_at) VALUES (?, ?, ?)",
		groupID[:], userID[:], time.Now(),
	)
	return err
}

func (s *SQLiteStorage) RemoveGroupMember(groupID, userID [16]byte) error {
	_, err := s.db.Exec(
		"DELETE FROM group_members WHERE group_id = ? AND user_id = ?",
		groupID[:], userID[:],
	)
	return err
}

func (s *SQLiteStorage) GetGroupMembers(groupID [16]byte) ([]GroupMemberEntry, error) {
	rows, err := s.db.Query(`
		SELECT u.id, u.username, k.public_key
		FROM group_members gm
		JOIN users u ON u.id = gm.user_id
		JOIN user_keys k ON k.user_id = u.id
		WHERE gm.group_id = ?`,
		groupID[:],
	)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения участников группы: %w", err)
	}
	defer rows.Close()

	var result []GroupMemberEntry
	for rows.Next() {
		var m GroupMemberEntry
		var idBytes, pubKeyBytes []byte
		if err := rows.Scan(&idBytes, &m.Username, &pubKeyBytes); err != nil {
			return nil, fmt.Errorf("ошибка чтения участника: %w", err)
		}
		if len(idBytes) != 16 || len(pubKeyBytes) != 32 {
			continue
		}
		copy(m.UserID[:], idBytes)
		copy(m.PublicKey[:], pubKeyBytes)
		result = append(result, m)
	}
	return result, rows.Err()
}

func (s *SQLiteStorage) IsGroup(id [16]byte) (bool, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM groups WHERE id = ?", id[:]).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (s *SQLiteStorage) SaveGroupKey(groupID, userID [16]byte, encryptedKey []byte) error {
	_, err := s.db.Exec(
		"INSERT OR REPLACE INTO group_keys (group_id, user_id, encrypted_key) VALUES (?, ?, ?)",
		groupID[:], userID[:], encryptedKey,
	)
	return err
}

func (s *SQLiteStorage) GetGroupKey(groupID, userID [16]byte) ([]byte, error) {
	row := s.db.QueryRow(
		"SELECT encrypted_key FROM group_keys WHERE group_id = ? AND user_id = ?",
		groupID[:], userID[:],
	)
	var key []byte
	err := row.Scan(&key)
	if err != nil {
		return nil, err
	}
	return key, nil
}

func (s *SQLiteStorage) DeleteGroup(groupID [16]byte) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec("DELETE FROM group_keys WHERE group_id = ?", groupID[:]); err != nil {
		return err
	}
	if _, err := tx.Exec("DELETE FROM group_members WHERE group_id = ?", groupID[:]); err != nil {
		return err
	}
	if _, err := tx.Exec("DELETE FROM groups WHERE id = ?", groupID[:]); err != nil {
		return err
	}
	return tx.Commit()
}
