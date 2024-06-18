package dbtypes

type AssignedSlot struct {
	Slot     uint64 `db:"slot"`
	Proposer uint64 `db:"proposer"`
	Block    *Slot  `db:"block"`
}

type BlockStatus struct {
	Root   []byte     `db:"root"`
	Status SlotStatus `db:"status"`
}

type AssignedBlob struct {
	Root       []byte `db:"root"`
	Commitment []byte `db:"commitment"`
	Slot       uint64 `db:"slot"`
	Blob       *Blob  `db:"blob"`
}

type BlockFilter struct {
	Graffiti      string
	ExtraData     string
	ProposerIndex *uint64
	ProposerName  string
	WithOrphaned  uint8
	WithMissing   uint8
}

type MevBlockFilter struct {
	MinSlot       uint64
	MaxSlot       uint64
	MinIndex      uint64
	MaxIndex      uint64
	ProposerName  string
	BuilderPubkey []byte
	Proposed      []uint8
	MevRelay      []uint8
}

type DepositTxFilter struct {
	Address       []byte
	TargetAddress []byte
	PublicKey     []byte
	ValidatorName string
	MinAmount     uint64
	MaxAmount     uint64
	WithOrphaned  uint8
	WithValid     uint8
}

type DepositFilter struct {
	MinIndex      uint64
	MaxIndex      uint64
	PublicKey     []byte
	ValidatorName string
	MinAmount     uint64
	MaxAmount     uint64
	WithOrphaned  uint8
}

type VoluntaryExitFilter struct {
	MinSlot       uint64
	MaxSlot       uint64
	MinIndex      uint64
	MaxIndex      uint64
	ValidatorName string
	WithOrphaned  uint8
}

type SlashingFilter struct {
	MinSlot       uint64
	MaxSlot       uint64
	MinIndex      uint64
	MaxIndex      uint64
	ValidatorName string
	SlasherName   string
	WithOrphaned  uint8
	WithReason    SlashingReason
}
