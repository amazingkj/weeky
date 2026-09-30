package repository

import (
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"testing"
)

func TestCreateUserWithInviteCode_ConcurrentUseOnlyOnce(t *testing.T) {
	repo, err := New(filepath.Join(t.TempDir(), "invite.db"))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer repo.Close()

	admin, err := repo.CreateUser("admin@test.com", "hash", "admin", true)
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	if _, err := repo.CreateInviteCode("CODE1", admin.ID); err != nil {
		t.Fatalf("CreateInviteCode: %v", err)
	}

	const n = 10
	var wg sync.WaitGroup
	errs := make([]error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, errs[i] = repo.CreateUserWithInviteCode("CODE1", fmt.Sprintf("u%d@test.com", i), "hash", "user")
		}(i)
	}
	wg.Wait()

	success := 0
	for _, e := range errs {
		switch {
		case e == nil:
			success++
		case errors.Is(e, ErrInviteCodeUsed):
		default:
			t.Fatalf("unexpected error: %v", e)
		}
	}
	if success != 1 {
		t.Fatalf("초대 코드 1개로 %d명 가입됨, 1명이어야 함", success)
	}

	// 실패한 요청의 사용자는 롤백되어 남지 않아야 함
	count, err := repo.CountUsers()
	if err != nil {
		t.Fatalf("CountUsers: %v", err)
	}
	if count != 2 {
		t.Fatalf("users = %d, want 2 (admin + 1)", count)
	}
}
