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

func TestReleaseBumpsGeneration(t *testing.T) {
	pool := New(2)
	handle, _ := pool.Spawn(Entity{Kind: "bullet", ID: 1})
	before := pool.GenerationOf(handle.Index)
	if !pool.Release(handle) {
		t.Fatal("首次归还应该成功")
	}
	if after := pool.GenerationOf(handle.Index); after != before+1 {
		t.Fatalf("回收后代次应该加一：before=%d after=%d", before, after)
	}
}

func TestStaleHandleIsRejectedAfterRecycle(t *testing.T) {
	pool := New(1)
	old, _ := pool.Spawn(Entity{Kind: "bullet", ID: 1})
	pool.Release(old)
	fresh, ok := pool.Spawn(Entity{Kind: "bullet", ID: 2})
	if !ok || fresh.Index != old.Index {
		t.Fatalf("回收后新实体应该复用同一个槽：%+v %v", fresh, ok)
	}
	if fresh.Generation == old.Generation {
		t.Fatal("复用时新句柄的代次应该不同")
	}
	if entity, ok := pool.Get(old); ok {
		t.Fatalf("过期句柄必须被拒绝，却读到了实体 ID=%d", entity.ID)
	}
	if entity, ok := pool.Get(fresh); !ok || entity.ID != 2 {
		t.Fatalf("新句柄应该正常访问：%+v %v", entity, ok)
	}
}

func TestDoubleReleaseIsRejected(t *testing.T) {
	pool := New(4)
	handle, _ := pool.Spawn(Entity{Kind: "bullet", ID: 1})
	if !pool.Release(handle) {
		t.Fatal("首次归还应该成功")
	}
	if pool.Release(handle) {
		t.Fatal("重复归还同一个句柄必须被拒绝")
	}
	if pool.Alive() != 0 || pool.Free() != pool.Capacity() {
		t.Fatalf("重复归还不该改计数：alive=%d free=%d", pool.Alive(), pool.Free())
	}
	if !pool.Consistent() {
		t.Fatal("重复归还被拒绝后池仍然应该自洽")
	}
}

func TestStaleHandleCannotReleaseReusedSlot(t *testing.T) {
	pool := New(1)
	old, _ := pool.Spawn(Entity{Kind: "bullet", ID: 1})
	pool.Release(old)
	fresh, _ := pool.Spawn(Entity{Kind: "bullet", ID: 2})
	if pool.Release(old) {
		t.Fatal("过期句柄不能回收已被别人占用的槽")
	}
	if pool.Alive() != 1 {
		t.Fatalf("新实体不该被旧句柄误回收：alive=%d", pool.Alive())
	}
	if entity, ok := pool.Get(fresh); !ok || entity.ID != 2 {
		t.Fatalf("新实体必须仍然活着：%+v %v", entity, ok)
	}
}

func TestOutOfRangeReleaseIsRejected(t *testing.T) {
	pool := New(2)
	if pool.Release(Handle{Index: 9, Generation: 0}) {
		t.Fatal("越界下标归还必须被拒绝")
	}
}

func TestFreeStackNeverContainsDuplicates(t *testing.T) {
	pool := New(16)
	var live []Handle
	for round := 0; round < 5000; round++ {
		switch round % 3 {
		case 0, 1:
			if handle, ok := pool.Spawn(Entity{Kind: "bullet", ID: round}); ok {
				live = append(live, handle)
			}
		case 2:
			if len(live) > 0 {
				pool.Release(live[0])
				live = live[1:]
			}
		}
		if pool.Alive()+pool.Free() != pool.Capacity() {
			t.Fatalf("第 %d 轮：alive+free 不等于容量：%d+%d", round, pool.Alive(), pool.Free())
		}
		if !pool.Consistent() {
			t.Fatalf("第 %d 轮：池不自洽（空闲栈可能有重复下标）", round)
		}
	}
}

func TestLongRunConservation(t *testing.T) {
	const capacity = 4096
	const iterations = 1_000_000
	pool := New(capacity)
	live := make([]Handle, 0, capacity)
	spawned, released := 0, 0
	for index := 0; index < iterations; index++ {
		if len(live) == capacity {
			if !pool.Release(live[0]) {
				t.Fatal("归还在场句柄必须成功")
			}
			live = live[1:]
			released++
		}
		handle, ok := pool.Spawn(Entity{Kind: "bullet", ID: index})
		if !ok {
			t.Fatal("刚腾出槽位就应该能生成")
		}
		live = append(live, handle)
		spawned++
		if index%50_000 == 0 {
			if pool.Free() > capacity || pool.Alive() < 0 {
				t.Fatalf("第 %d 次迭代计数越界：alive=%d free=%d", index, pool.Alive(), pool.Free())
			}
			if !pool.Consistent() {
				t.Fatalf("第 %d 次迭代池不自洽", index)
			}
		}
	}
	if spawned-released != pool.Alive() {
		t.Fatalf("spawned-released 不等于 alive：%d-%d != %d", spawned, released, pool.Alive())
	}
	if pool.Free() != capacity-pool.Alive() {
		t.Fatalf("free 不等于 capacity-alive：%d != %d", pool.Free(), capacity-pool.Alive())
	}
	if !pool.Consistent() {
		t.Fatal("百万次迭代后池不自洽")
	}
}
