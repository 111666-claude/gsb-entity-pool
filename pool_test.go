package gamepool

import "testing"

func TestSpawnAndGet(t *testing.T) {
	pool := New(4)
	handle, ok := pool.Spawn(Entity{Kind: "bullet", ID: 7, X: 1, Life: 5})
	if !ok {
		t.Fatal("空池应该能生成")
	}
	entity, ok := pool.Get(handle)
	if !ok || entity.ID != 7 || entity.Kind != "bullet" {
		t.Fatalf("取回的实体不对：%+v %v", entity, ok)
	}
}

func TestCapacityIsRespected(t *testing.T) {
	pool := New(3)
	for index := 0; index < 3; index++ {
		if _, ok := pool.Spawn(Entity{Kind: "bullet", ID: index}); !ok {
			t.Fatalf("第 %d 次生成不该失败", index)
		}
	}
	if _, ok := pool.Spawn(Entity{Kind: "bullet", ID: 99}); ok {
		t.Fatal("池满之后不该再生成成功")
	}
}

func TestReleaseLetsSlotBeReused(t *testing.T) {
	pool := New(1)
	handle, _ := pool.Spawn(Entity{Kind: "bullet", ID: 1})
	pool.Release(handle)
	again, ok := pool.Spawn(Entity{Kind: "bullet", ID: 2})
	if !ok || again.Index != handle.Index {
		t.Fatalf("回收后应该能复用同一个槽：%+v %v", again, ok)
	}
}

func TestGetRejectsOutOfRange(t *testing.T) {
	pool := New(2)
	if _, ok := pool.Get(Handle{Index: 9}); ok {
		t.Fatal("越界下标应该被拒绝")
	}
}

func TestAliveCountsSpawns(t *testing.T) {
	pool := New(4)
	pool.Spawn(Entity{Kind: "bullet"})
	pool.Spawn(Entity{Kind: "bullet"})
	if pool.Alive() != 2 {
		t.Fatalf("在场实体数不对：%d", pool.Alive())
	}
}
