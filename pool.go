// Package gamepool 是游戏实体池：子弹复用固定容量的槽位。
//
// 槽位带代次（generation）：每次回收时代次加一，句柄必须同时匹配
// 下标与代次才能访问槽位。inUse 占用位让 Spawn/Release/Get 全部 O(1)，
// 重复归还、过期句柄都会被直接拒绝，空闲栈不可能出现重复下标。
package gamepool

// Handle 是实体句柄：下标 + 槽位代次。
type Handle struct {
	Index      uint32
	Generation uint32
}

// Entity 是一个实体。
type Entity struct {
	Kind string
	ID   int
	X    float64
	Y    float64
	VX   float64
	VY   float64
	Life int
}

// Pool 是固定容量的实体池。
type Pool struct {
	entries     []Entity
	generations []uint32
	inUse       []bool
	free        []uint32
	alive       int
}

// New 建池。
func New(capacity int) *Pool {
	if capacity < 1 {
		capacity = 1
	}
	pool := &Pool{
		entries:     make([]Entity, capacity),
		generations: make([]uint32, capacity),
		inUse:       make([]bool, capacity),
	}
	for index := capacity - 1; index >= 0; index-- {
		pool.free = append(pool.free, uint32(index))
	}
	return pool
}

// Spawn 取一个空槽。池满时返回 false（由调用方计入丢弃统计）。
func (p *Pool) Spawn(entity Entity) (Handle, bool) {
	if len(p.free) == 0 {
		return Handle{}, false
	}
	index := p.free[len(p.free)-1]
	p.free = p.free[:len(p.free)-1]
	p.entries[index] = entity
	p.inUse[index] = true
	p.alive++
	return Handle{Index: index, Generation: p.generations[index]}, true
}

// Get 同时比对下标与代次：越界、空闲槽、过期句柄一律拒绝。
func (p *Pool) Get(handle Handle) (*Entity, bool) {
	if int(handle.Index) >= len(p.entries) {
		return nil, false
	}
	if !p.inUse[handle.Index] || p.generations[handle.Index] != handle.Generation {
		return nil, false
	}
	return &p.entries[handle.Index], true
}

// Release 回收槽位。只有「下标有效、槽位在用、代次匹配」的句柄才能回收：
// 重复归还、过期句柄、越界下标都返回 false，不入栈也不动计数。
// 回收成功时代次加一，使指向该槽的旧句柄全部失效。
func (p *Pool) Release(handle Handle) bool {
	if int(handle.Index) >= len(p.entries) {
		return false
	}
	if !p.inUse[handle.Index] || p.generations[handle.Index] != handle.Generation {
		return false
	}
	p.inUse[handle.Index] = false
	p.generations[handle.Index]++
	p.free = append(p.free, handle.Index)
	p.alive--
	return true
}

// GenerationOf 是诊断接口：读某个槽当前的代次。
func (p *Pool) GenerationOf(index uint32) uint32 {
	if int(index) >= len(p.generations) {
		return 0
	}
	return p.generations[index]
}

// Capacity 是槽位数。
func (p *Pool) Capacity() int {
	return len(p.entries)
}

// Alive 是在场实体数。
func (p *Pool) Alive() int {
	return p.alive
}

// Free 是空闲栈的长度。
func (p *Pool) Free() int {
	return len(p.free)
}

// Consistent 是诊断接口：检查计数、空闲栈重复下标以及占用位与计数是否自洽。
func (p *Pool) Consistent() bool {
	if p.alive < 0 || p.alive > len(p.entries) || len(p.free) > len(p.entries) {
		return false
	}
	if p.alive+len(p.free) != len(p.entries) {
		return false
	}
	seen := make([]bool, len(p.entries))
	used := 0
	for _, inUse := range p.inUse {
		if inUse {
			used++
		}
	}
	if used != p.alive {
		return false
	}
	for _, index := range p.free {
		if int(index) >= len(p.entries) || p.inUse[index] || seen[index] {
			return false
		}
		seen[index] = true
	}
	return true
}
