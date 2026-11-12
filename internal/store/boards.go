package store

import (
	"context"
	"errors"
	"fmt"
	"uuid"

	"github.com/jackc/pgx/v5"
	"gitverse.ru/zagvozdeen/agenda/internal/models"
	"gitverse.ru/zagvozdeen/agenda/internal/models/enums"
)

var (
	ErrBoardNotFound       = errors.New("board not found")
	ErrBoardMemberNotFound = errors.New("board member not found")
)

const boardColumns = "key, name, description, is_private, task_seq, sprint_seq, created_by, created_at, updated_at, deleted_at"

func (s *Store) GetBoard(ctx context.Context, key string, lock bool) (*models.Board, error) {
	query := "SELECT " + boardColumns + " FROM boards WHERE key = $1 AND deleted_at IS NULL"
	if lock {
		query += " FOR UPDATE"
	}
	board := &models.Board{}
	err := s.querier(ctx).QueryRow(
		ctx,
		query,
		key,
	).Scan(&board.Key, &board.Name, &board.Description, &board.IsPrivate, &board.TaskSeq, &board.SprintSeq, &board.CreatedBy, &board.CreatedAt, &board.UpdatedAt, &board.DeletedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrBoardNotFound
		}
		return nil, fmt.Errorf("failed to get board: %w", err)
	}
	return board, nil
}

func (s *Store) ListBoards(ctx context.Context, memberID, viewerID *uuid.UUID, limit, offset int) ([]models.Board, error) {
	rows, err := s.querier(ctx).Query(
		ctx,
		"SELECT "+boardColumns+" FROM boards AS b WHERE deleted_at IS NULL AND ($1::uuid IS NULL OR EXISTS (SELECT 1 FROM board_members AS m WHERE m.board_key = b.key AND m.user_id = $1)) AND (NOT b.is_private OR EXISTS (SELECT 1 FROM board_members AS v WHERE v.board_key = b.key AND v.user_id = $4)) ORDER BY key LIMIT $2 OFFSET $3",
		memberID, limit, offset, viewerID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list boards: %w", err)
	}
	defer rows.Close()
	var boards []models.Board
	for rows.Next() {
		var board models.Board
		err = rows.Scan(&board.Key, &board.Name, &board.Description, &board.IsPrivate, &board.TaskSeq, &board.SprintSeq, &board.CreatedBy, &board.CreatedAt, &board.UpdatedAt, &board.DeletedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to list boards: %w", err)
		}
		boards = append(boards, board)
	}
	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("failed to list boards: %w", err)
	}
	return boards, nil
}

func (s *Store) ListMemberships(ctx context.Context, userID uuid.UUID) ([]models.Membership, error) {
	rows, err := s.querier(ctx).Query(
		ctx,
		"SELECT m.board_key, b.name, m.role FROM board_members AS m JOIN boards AS b ON b.key = m.board_key WHERE m.user_id = $1 AND b.deleted_at IS NULL ORDER BY m.board_key",
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list memberships: %w", err)
	}
	defer rows.Close()
	var memberships []models.Membership
	for rows.Next() {
		var membership models.Membership
		err = rows.Scan(&membership.BoardKey, &membership.Name, &membership.Role)
		if err != nil {
			return nil, fmt.Errorf("failed to list memberships: %w", err)
		}
		memberships = append(memberships, membership)
	}
	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("failed to list memberships: %w", err)
	}
	return memberships, nil
}

func (s *Store) CreateBoard(ctx context.Context, board *models.Board) error {
	err := s.querier(ctx).QueryRow(
		ctx,
		"INSERT INTO boards (key, name, description, is_private, created_by, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, now(), now()) RETURNING created_at, updated_at",
		board.Key, board.Name, board.Description, board.IsPrivate, board.CreatedBy,
	).Scan(&board.CreatedAt, &board.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create board: %w", err)
	}
	return nil
}

func (s *Store) UpdateBoard(ctx context.Context, board *models.Board) error {
	err := s.querier(ctx).QueryRow(
		ctx,
		"UPDATE boards SET name = $2, description = $3, is_private = $4, updated_at = now() WHERE key = $1 RETURNING updated_at",
		board.Key, board.Name, board.Description, board.IsPrivate,
	).Scan(&board.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrBoardNotFound
		}
		return fmt.Errorf("failed to update board: %w", err)
	}
	return nil
}

func (s *Store) DeleteBoard(ctx context.Context, key string) error {
	_, err := s.querier(ctx).Exec(
		ctx,
		"UPDATE boards SET deleted_at = now(), updated_at = now() WHERE key = $1",
		key,
	)
	if err != nil {
		return fmt.Errorf("failed to delete board: %w", err)
	}
	return nil
}

func (s *Store) GetBoardRole(ctx context.Context, key string, userID uuid.UUID) (enums.BoardRole, error) {
	var role enums.BoardRole
	err := s.querier(ctx).QueryRow(
		ctx,
		"SELECT role FROM board_members WHERE board_key = $1 AND user_id = $2",
		key, userID,
	).Scan(&role)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrBoardMemberNotFound
		}
		return "", fmt.Errorf("failed to get board role: %w", err)
	}
	return role, nil
}

func (s *Store) CountOwners(ctx context.Context, key string) (int, error) {
	var count int
	err := s.querier(ctx).QueryRow(
		ctx,
		"SELECT count(*) FROM board_members WHERE board_key = $1 AND role = 'owner'",
		key,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count board owners: %w", err)
	}
	return count, nil
}

func (s *Store) ListMembers(ctx context.Context, key string) ([]models.Member, error) {
	rows, err := s.querier(ctx).Query(
		ctx,
		"SELECT m.board_key, m.user_id, m.role, m.added_at, u.username, u.name, u.deleted_at IS NOT NULL AS disabled FROM board_members AS m JOIN users AS u ON u.id = m.user_id WHERE m.board_key = $1 ORDER BY m.role, u.username",
		key,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list board members: %w", err)
	}
	defer rows.Close()
	var members []models.Member
	for rows.Next() {
		var member models.Member
		err = rows.Scan(&member.BoardKey, &member.UserID, &member.Role, &member.AddedAt, &member.Username, &member.Name, &member.Disabled)
		if err != nil {
			return nil, fmt.Errorf("failed to list board members: %w", err)
		}
		members = append(members, member)
	}
	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("failed to list board members: %w", err)
	}
	return members, nil
}

func (s *Store) SaveMember(ctx context.Context, member *models.BoardMember) error {
	err := s.querier(ctx).QueryRow(
		ctx,
		"INSERT INTO board_members (board_key, user_id, role, added_at) VALUES ($1, $2, $3, now()) ON CONFLICT (board_key, user_id) DO UPDATE SET role = EXCLUDED.role RETURNING added_at",
		member.BoardKey, member.UserID, member.Role,
	).Scan(&member.AddedAt)
	if err != nil {
		return fmt.Errorf("failed to save board member: %w", err)
	}
	return nil
}

func (s *Store) RemoveMember(ctx context.Context, key string, userID uuid.UUID) error {
	_, err := s.querier(ctx).Exec(
		ctx,
		"DELETE FROM board_members WHERE board_key = $1 AND user_id = $2",
		key, userID,
	)
	if err != nil {
		return fmt.Errorf("failed to remove board member: %w", err)
	}
	return nil
}

func (s *Store) CountTasksByStatus(ctx context.Context, key string) (map[enums.TaskStatus]int, error) {
	rows, err := s.querier(ctx).Query(
		ctx,
		"SELECT status, count(*) FROM tasks WHERE board_key = $1 AND deleted_at IS NULL GROUP BY status",
		key,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to count tasks by status: %w", err)
	}
	defer rows.Close()
	counts := map[enums.TaskStatus]int{enums.TaskStatusTodo: 0, enums.TaskStatusInProgress: 0, enums.TaskStatusReview: 0, enums.TaskStatusDone: 0, enums.TaskStatusCancelled: 0}
	for rows.Next() {
		var status enums.TaskStatus
		var count int
		err = rows.Scan(&status, &count)
		if err != nil {
			return nil, fmt.Errorf("failed to count tasks by status: %w", err)
		}
		counts[status] = count
	}
	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("failed to count tasks by status: %w", err)
	}
	return counts, nil
}
