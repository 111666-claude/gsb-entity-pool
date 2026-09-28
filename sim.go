// 包 gamepool 的帧循环：子弹生成、移动、到期，特效跟随。
//
// 子弹存放在实体池里；特效只是死亡子弹的短暂视觉残留，不再占用池槽位，
// 也不持有可写的池实体。特效跟随旧主人时必须通过代次校验，旧句柄在
// 槽位回收的一刻就失效，因此残留数据不可能写到复用该槽的新子弹身上。
package gamepool

// Stats 是仿真统计。
type Stats struct {
	Frames      int
	Spawned     int
	Dropped     int
	Released    int
	Alive       int
	Free        int
	GhostWrites int
	Consistent  bool
}

// Sim 是单线程的游戏循环。
type effectState struct {
	owner   Handle
	ownerID int
	x       float64
	y       float64
	life    int
}

type Sim struct {
	pool    *Pool
	bullets map[Handle]bool
	effects map[int]effectState
	stats   Stats
	nextID  int
}

// NewSim 建仿真。
func NewSim(capacity int) *Sim {
	return &Sim{
		pool:    New(capacity),
		bullets: map[Handle]bool{},
		effects: map[int]effectState{},
	}
}

// Pool 返回底层实体池（诊断用）。
func (s *Sim) Pool() *Pool {
	return s.pool
}

// Stats 返回当前统计。
func (s *Sim) Stats() Stats {
	snapshot := s.stats
	snapshot.Alive = s.pool.Alive()
	snapshot.Free = s.pool.Free()
	snapshot.Consistent = s.pool.Consistent()
	return snapshot
}

// Step 走一帧：先推进老子弹并回收到期的，再生成新子弹，最后更新特效。
func (s *Sim) Step(spawnPerFrame int) {
	s.stats.Frames++

	for handle := range s.bullets {
		entity, ok := s.pool.Get(handle)
		if !ok {
			// 句柄已失效（正常情况下不会发生），只清理本地登记。
			delete(s.bullets, handle)
			continue
		}
		entity.X += entity.VX
		entity.Y += entity.VY
		entity.Life--
		if entity.Life > 0 {
			continue
		}
		bulletID := entity.ID
		deathX, deathY := entity.X, entity.Y
		if s.pool.Release(handle) {
			s.stats.Released++
		}
		delete(s.bullets, handle)
		s.nextID++
		s.effects[s.nextID] = effectState{
			owner:   handle, // 已随回收失效，特效只能用它做只读的代次校验
			ownerID: bulletID,
			x:       deathX,
			y:       deathY,
			life:    5,
		}
	}

	for index := 0; index < spawnPerFrame; index++ {
		s.nextID++
		handle, ok := s.pool.Spawn(Entity{
			Kind: "bullet", ID: s.nextID,
			X: -20, Y: float64(index), VX: 3, Life: 30,
		})
		if !ok {
			s.stats.Dropped++
			continue
		}
		s.bullets[handle] = true
		s.stats.Spawned++
	}

	for id, effect := range s.effects {
		effect.life--
		effect.x += 0.5
		// 跟随旧主人：句柄在子弹回收时已经换代，Get 必失败，
		// 因此不会有任何写入落到复用该槽的新子弹身上。
		if ownerEntity, ok := s.pool.Get(effect.owner); ok {
			ownerEntity.X += 0.5
			if ownerEntity.ID != effect.ownerID {
				s.stats.GhostWrites++
			}
		}
		if effect.life <= 0 {
			delete(s.effects, id)
			continue
		}
		s.effects[id] = effect
	}
}

// Run 跑若干帧。
func (s *Sim) Run(frames, spawnPerFrame int) Stats {
	for index := 0; index < frames; index++ {
		s.Step(spawnPerFrame)
	}
	return s.Stats()
}

// StaleCheck 是一次专门测过期句柄的小场景：生成→回收→再生成→用旧句柄访问。
func StaleCheck() (accepted bool, generationChanged bool) {
	pool := New(4)
	first, ok := pool.Spawn(Entity{Kind: "bullet", ID: 1, Life: 10})
	if !ok {
		return false, false
	}
	pool.Release(first)
	second, ok := pool.Spawn(Entity{Kind: "bullet", ID: 2, Life: 10})
	if !ok {
		return false, false
	}
	if second.Index != first.Index {
		return false, false
	}
	_, accepted = pool.Get(first)
	generationChanged = pool.GenerationOf(first.Index) != first.Generation
	return accepted, generationChanged
}
