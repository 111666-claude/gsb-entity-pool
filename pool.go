// Package gamepool 是游戏实体池：子弹与特效复用固定容量的槽位。
//
// 句柄带代次：槽位每回收一次 Generation 加一，Get 同时比对下标与代次，
// 过期句柄一律拒绝。每个槽位还有一张在/闲位图，Release 只能归还
// 「当前确实在用」的槽，重复归还直接忽略，因此空闲栈不会出现重复下标。
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
	inuse       []bool
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
		inuse:       make([]bool, capacity),
	}
	// 下标倒序压栈，弹出时先拿到 0，行为直观且与既有测试一致。
	pool.free = make([]uint32, 0, capacity)
	for index := capacity - 1; index >= 0; index-- {
		pool.free = append(pool.free, uint32(index))
	}
	return pool
}

// Spawn 取一个空槽，O(1)。池满时返回 false，由调用方记 dropped。
func (p *Pool) Spawn(entity Entity) (Handle, bool) {
	if len(p.free) == 0 {
		return Handle{}, false
	}
	last := len(p.free) - 1
	index := p.free[last]
	p.free = p.free[:last]
	p.inuse[index] = true
	p.entries[index] = entity
	p.alive++
	return Handle{Index: index, Generation: p.generations[index]}, true
}

// Get 取回句柄对应的实体，O(1)。下标越界、槽位空闲或代次过期都拒绝。
func (p *Pool) Get(handle Handle) (*Entity, bool) {
	index := int(handle.Index)
	if index >= len(p.entries) {
		return nil, false
	}
	if !p.inuse[index] || p.generations[index] != handle.Generation {
		return nil, false
	}
	return &p.entries[index], true
}

// Release 归还一个槽，O(1)。只接受「在用且代次匹配」的句柄：
// 已经在空闲栈里的槽重复归还时直接返回 false，不再入栈、不重复减计数。
func (p *Pool) Release(handle Handle) bool {
	index := int(handle.Index)
	if index >= len(p.entries) {
		return false
	}
	if !p.inuse[index] || p.generations[index] != handle.Generation {
		return false
	}
	p.inuse[index] = false
	p.generations[index]++
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

// Consistent 是诊断接口：检查计数边界、空闲栈有无重复/越界下标，
// 以及在/闲位图、代次、计数三者是否自洽。
// 仅用于校验（如测试），不在 Spawn/Release 的热路径上。
func (p *Pool) Consistent() bool {
	if p.alive < 0 || p.alive > len(p.entries) || len(p.free) > len(p.entries) {
		return false
	}
	if p.alive+len(p.free) != len(p.entries) {
		return false
	}
	seen := make([]bool, len(p.entries))
	for _, index := range p.free {
		if int(index) >= len(p.entries) || seen[index] || p.inuse[index] {
			return false
		}
		seen[index] = true
	}
	inUseCount := 0
	for _, active := range p.inuse {
		if active {
			inUseCount++
		}
	}
	return inUseCount == p.alive
}
