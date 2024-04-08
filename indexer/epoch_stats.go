package indexer

import (
	"bytes"
	"fmt"
	"math"
	"sync"

	v1 "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/attestantio/go-eth2-client/spec/phase0"

	"github.com/ethpandaops/dora/db"
	"github.com/ethpandaops/dora/dbtypes"
	"github.com/ethpandaops/dora/rpc"
	"github.com/ethpandaops/dora/utils"
)

type EpochStats struct {
	Epoch               uint64
	DependentRoot       []byte
	dependentStateRef   string
	seenCount           uint64
	isInDb              bool
	dutiesMutex         sync.RWMutex
	validatorsMutex     sync.RWMutex
	proposerAssignments map[uint64]uint64
	attestorAssignments map[string][]uint64
	syncAssignments     []uint64
	validatorStats      *EpochValidatorStats
	dbEpochMutex        sync.Mutex
	dbEpochCache        *dbtypes.Epoch
}

type EpochValidatorStats struct {
	ValidatorCount    uint64
	ValidatorBalance  uint64
	EligibleAmount    uint64
	ValidatorBalances map[uint64]uint64
}

func (cache *indexerCache) getEpochStats(epoch uint64, dependendRoot []byte) *EpochStats {
	cache.epochStatsMutex.RLock()
	defer cache.epochStatsMutex.RUnlock()
	if cache.epochStatsMap[epoch] != nil {
		for _, epochStats := range cache.epochStatsMap[epoch] {
			if dependendRoot == nil || bytes.Equal(epochStats.DependentRoot, dependendRoot) {
				return epochStats
			}
		}
	}
	return nil
}

func (cache *indexerCache) createOrGetEpochStats(epoch uint64, dependendRoot []byte) (*EpochStats, bool) {
	cache.epochStatsMutex.Lock()
	defer cache.epochStatsMutex.Unlock()
	if cache.epochStatsMap[epoch] == nil {
		cache.epochStatsMap[epoch] = make([]*EpochStats, 0)
	} else {
		for _, epochStats := range cache.epochStatsMap[epoch] {
			if bytes.Equal(epochStats.DependentRoot, dependendRoot) {
				return epochStats, false
			}
		}
	}
	epochStats := &EpochStats{
		Epoch:         epoch,
		DependentRoot: dependendRoot,
	}
	cache.epochStatsMap[epoch] = append(cache.epochStatsMap[epoch], epochStats)
	return epochStats, true
}

func (cache *indexerCache) removeEpochStats(epochStats *EpochStats) {
	cache.epochStatsMutex.Lock()
	defer cache.epochStatsMutex.Unlock()
	logger.Debugf("remove cached epoch stats: %v", epochStats.Epoch)

	allEpochStats := cache.epochStatsMap[epochStats.Epoch]
	if allEpochStats != nil {
		var idx uint64
		len := uint64(len(allEpochStats))
		for idx = 0; idx < len; idx++ {
			if allEpochStats[idx] == epochStats {
				break
			}
		}
		if idx < len {
			if len == 1 {
				delete(cache.epochStatsMap, epochStats.Epoch)
			} else {
				if idx < len-1 {
					cache.epochStatsMap[epochStats.Epoch][idx] = cache.epochStatsMap[epochStats.Epoch][len-1]
				}
				cache.epochStatsMap[epochStats.Epoch] = cache.epochStatsMap[epochStats.Epoch][0 : len-1]
			}
		}
	}
}

func (epochStats *EpochStats) IsReady() bool {
	if !epochStats.dutiesMutex.TryRLock() {
		return false
	}
	epochStats.dutiesMutex.RUnlock()
	return true
}

func (epochStats *EpochStats) IsValidatorsReady() bool {
	if !epochStats.validatorsMutex.TryRLock() {
		return false
	}
	epochStats.validatorsMutex.RUnlock()
	return true
}

func (epochStats *EpochStats) GetDependentStateRef() string {
	epochStats.dutiesMutex.RLock()
	defer epochStats.dutiesMutex.RUnlock()
	return epochStats.dependentStateRef
}

func (epochStats *EpochStats) GetProposerAssignments() map[uint64]uint64 {
	epochStats.dutiesMutex.RLock()
	defer epochStats.dutiesMutex.RUnlock()
	return epochStats.proposerAssignments
}

func (epochStats *EpochStats) TryGetProposerAssignments() map[uint64]uint64 {
	if !epochStats.dutiesMutex.TryRLock() {
		return nil
	}
	defer epochStats.dutiesMutex.RUnlock()
	return epochStats.proposerAssignments
}

func (epochStats *EpochStats) GetAttestorAssignments() map[string][]uint64 {
	epochStats.dutiesMutex.RLock()
	defer epochStats.dutiesMutex.RUnlock()
	return epochStats.attestorAssignments
}

func (epochStats *EpochStats) GetSyncAssignments() []uint64 {
	epochStats.dutiesMutex.RLock()
	defer epochStats.dutiesMutex.RUnlock()
	return epochStats.syncAssignments
}

func (epochStats *EpochStats) TryGetSyncAssignments() []uint64 {
	if !epochStats.dutiesMutex.TryRLock() {
		return nil
	}
	defer epochStats.dutiesMutex.RUnlock()
	return epochStats.syncAssignments
}

func (client *IndexerClient) ensureEpochStats(epoch uint64, head []byte) error {
	var dependentRoot []byte
	var proposerRsp *rpc.ProposerDuties
	firstBlock := client.indexerCache.getFirstCanonicalBlock(epoch, head)
	if firstBlock != nil {
		logger.WithField("client", client.clientName).Tracef("canonical first block for epoch %v: %v/0x%x (head: 0x%x)", epoch, firstBlock.Slot, firstBlock.Root, head)
		if epoch == 0 {
			// dependent root for epoch 0 is genesis block
			dependentRoot = firstBlock.Root
		} else {
			dependentRoot = firstBlock.GetParentRoot()
		}
	}
	if dependentRoot == nil && epoch > 0 {
		lastBlock := client.indexerCache.getLastCanonicalBlock(epoch-1, head)
		if lastBlock != nil {
			logger.WithField("client", client.clientName).Tracef("canonical last block for epoch %v: %v/0x%x (head: 0x%x)", epoch-1, lastBlock.Slot, lastBlock.Root, head)
			dependentRoot = lastBlock.Root
		}
	}
	if dependentRoot == nil {
		if utils.Config.Chain.WhiskForkEpoch != nil && epoch >= *utils.Config.Chain.WhiskForkEpoch {
			firstSlot := epoch * utils.Config.Chain.Config.SlotsPerEpoch
			dependentRoot = db.GetHighestRootBeforeSlot(firstSlot, false)
		} else {
			var err error
			proposerRsp, err = client.rpcClient.GetProposerDuties(epoch)
			if err != nil {
				logger.WithField("client", client.clientName).Warnf("could not load proposer duties for epoch %v: %v", epoch, err)
				return fmt.Errorf("could not find proposer duties for epoch %v: %v", epoch, err)
			}
			if proposerRsp == nil {
				return fmt.Errorf("could not find proposer duties for epoch %v", epoch)
			}
			dependentRoot = proposerRsp.DependentRoot[:]
		}
	}

	epochStats, isNewStats := client.indexerCache.createOrGetEpochStats(epoch, dependentRoot)
	if isNewStats {
		logger.WithField("client", client.clientName).Infof("load epoch stats for epoch %v (dependend: 0x%x, head: 0x%x)", epoch, dependentRoot, head)
	} else {
		logger.WithField("client", client.clientName).Debugf("ensure epoch stats for epoch %v (dependend: 0x%x, head: 0x%x)", epoch, dependentRoot, head)
	}
	go func() {
		defer utils.HandleSubroutinePanic("ensureEpochStatsLazy")
		err := epochStats.ensureEpochStatsLazy(client, proposerRsp)
		if err != nil {
			logger.WithField("client", client.clientName).WithError(err).Warnf("error while loading epoch stats for epoch %v", epoch)
		}
	}()
	if int64(epoch) > client.lastEpochStats {
		client.lastEpochStats = int64(epoch)
	}
	return nil
}

func (epochStats *EpochStats) ensureEpochStatsLazy(client *IndexerClient, proposerRsp *rpc.ProposerDuties) error {
	epochStats.dutiesMutex.Lock()
	defer epochStats.dutiesMutex.Unlock()

	// proposer duties
	if epochStats.proposerAssignments == nil {
		whiskActivated := utils.Config.Chain.WhiskForkEpoch != nil && epochStats.Epoch >= *utils.Config.Chain.WhiskForkEpoch
		if proposerRsp == nil && !whiskActivated {
			var err error
			proposerRsp, err = client.rpcClient.GetProposerDuties(epochStats.Epoch)
			if err != nil {
				return fmt.Errorf("could not lazy load proposer duties for epoch %v: %v", epochStats.Epoch, err)
			}
			if proposerRsp == nil {
				return fmt.Errorf("could not find proposer duties for epoch %v", epochStats.Epoch)
			}
			if !bytes.Equal(proposerRsp.DependentRoot[:], epochStats.DependentRoot) {
				logger.WithField("client", client.clientName).Warnf("got unexpected dependend root for proposer duties %v (got: %v, expected: 0x%x)", epochStats.Epoch, proposerRsp.DependentRoot.String(), epochStats.DependentRoot)
				altEpochStats, isNewStats := client.indexerCache.createOrGetEpochStats(epochStats.Epoch, proposerRsp.DependentRoot[:])
				if isNewStats {
					logger.WithField("client", client.clientName).Infof("load epoch stats for epoch %v (dependend: 0x%x)", epochStats.Epoch, proposerRsp.DependentRoot[:])
				} else {
					logger.WithField("client", client.clientName).Debugf("ensure epoch stats for epoch %v (dependend: 0x%x)", epochStats.Epoch, proposerRsp.DependentRoot[:])
				}
				return altEpochStats.ensureEpochStatsLazy(client, proposerRsp)
			}
		}
		proposerAssignments := map[uint64]uint64{}
		if whiskActivated {
			firstSlot := epochStats.Epoch * utils.Config.Chain.Config.SlotsPerEpoch
			lastSlot := firstSlot + utils.Config.Chain.Config.SlotsPerEpoch - 1
			for slot := firstSlot; slot <= lastSlot; slot++ {
				proposerAssignments[slot] = math.MaxInt64
			}
		} else {
			for _, duty := range proposerRsp.Data {
				proposerAssignments[uint64(duty.Slot)] = uint64(duty.ValidatorIndex)
			}
		}
		epochStats.proposerAssignments = proposerAssignments
	}

	// get state root for dependend root
	if epochStats.dependentStateRef == "" {
		if epochStats.Epoch == 0 {
			epochStats.dependentStateRef = "genesis"
		} else if dependendBlock := client.indexerCache.getCachedBlock(epochStats.DependentRoot); dependendBlock != nil {
			if dependendBlock.Slot == 0 {
				epochStats.dependentStateRef = "genesis"
			} else {
				dependendBlock.mutex.RLock()
				epochStats.dependentStateRef = dependendBlock.header.Message.StateRoot.String()
				dependendBlock.mutex.RUnlock()
			}
		} else {
			parsedHeader, err := client.rpcClient.GetBlockHeaderByBlockroot(epochStats.DependentRoot)
			if err != nil {
				return fmt.Errorf("could not get dependent block header for epoch %v (0x%x): %v", epochStats.Epoch, epochStats.DependentRoot, err)
			}
			if parsedHeader == nil {
				return fmt.Errorf("could not get dependent block header for epoch %v (0x%x): not found", epochStats.Epoch, epochStats.DependentRoot)
			}
			if parsedHeader.Header.Message.Slot == 0 {
				epochStats.dependentStateRef = "genesis"
			} else {
				epochStats.dependentStateRef = parsedHeader.Header.Message.StateRoot.String()
			}
		}
	}

	// load validators
	if epochStats.validatorStats == nil {
		go epochStats.ensureValidatorStatsLazy(client, epochStats.dependentStateRef)
	}

	// get committee duties
	if epochStats.attestorAssignments == nil {
		parsedCommittees, err := client.rpcClient.GetCommitteeDuties(epochStats.dependentStateRef, epochStats.Epoch)
		if err != nil {
			return fmt.Errorf("error retrieving committees data: %v", err)
		}

		attestorAssignments := map[string][]uint64{}
		for _, committee := range parsedCommittees {
			for _, valIndex := range committee.Validators {
				valIndexU64 := uint64(valIndex)
				k := fmt.Sprintf("%v-%v", uint64(committee.Slot), uint64(committee.Index))
				if attestorAssignments[k] == nil {
					attestorAssignments[k] = make([]uint64, 0)
				}
				attestorAssignments[k] = append(attestorAssignments[k], valIndexU64)
			}
		}
		epochStats.attestorAssignments = attestorAssignments
	}

	// get sync committee duties
	if epochStats.syncAssignments == nil && epochStats.Epoch >= utils.Config.Chain.Config.AltairForkEpoch {
		syncCommitteeState := epochStats.dependentStateRef
		if epochStats.Epoch > 0 && epochStats.Epoch == utils.Config.Chain.Config.AltairForkEpoch {
			syncCommitteeState = fmt.Sprintf("%d", utils.Config.Chain.Config.AltairForkEpoch*utils.Config.Chain.Config.SlotsPerEpoch)
		}
		parsedSyncCommittees, err := client.rpcClient.GetSyncCommitteeDuties(syncCommitteeState, epochStats.Epoch)
		if err != nil {
			return fmt.Errorf("error retrieving sync_committees for epoch %v (state: %v): %v", epochStats.Epoch, syncCommitteeState, err)
		}
		if parsedSyncCommittees != nil {
			syncAssignments := make([]uint64, len(parsedSyncCommittees.Validators))
			for i, valIndex := range parsedSyncCommittees.Validators {
				valIndexU64 := uint64(valIndex)
				syncAssignments[i] = valIndexU64
			}
			epochStats.syncAssignments = syncAssignments
		}
	}

	epochStats.seenCount++

	return nil
}

func (epochStats *EpochStats) ensureValidatorStatsLazy(client *IndexerClient, stateRef string) {
	defer utils.HandleSubroutinePanic("ensureValidatorStatsLazy")
	if client.skipValidators {
		return
	}
	epochStats.loadValidatorStats(client, stateRef)
}

func (epochStats *EpochStats) loadValidatorStats(client *IndexerClient, stateRef string) {
	epochStats.validatorsMutex.Lock()
	defer epochStats.validatorsMutex.Unlock()
	if epochStats.validatorStats != nil {
		return
	}

	// `lock` concurrency limit (limit concurrent get validators calls)
	client.indexerCache.validatorLoadingLimiter <- 1

	var epochValidators map[phase0.ValidatorIndex]*v1.Validator
	var err error
	if epochStats.Epoch == 0 {
		epochValidators, err = client.rpcClient.GetStateValidators("genesis")
	} else {
		epochValidators, err = client.rpcClient.GetStateValidators(stateRef)
	}

	// `unlock` concurrency limit
	<-client.indexerCache.validatorLoadingLimiter

	if err != nil {
		logger.Warnf("error fetching epoch %v validators: %v", epochStats.Epoch, err)
		return
	}
	client.indexerCache.setLastValidators(epochStats.Epoch, epochValidators)
	validatorStats := &EpochValidatorStats{
		ValidatorBalances: make(map[uint64]uint64),
	}
	for _, validator := range epochValidators {
		validatorStats.ValidatorBalances[uint64(validator.Index)] = uint64(validator.Validator.EffectiveBalance)
		if uint64(validator.Validator.ActivationEpoch) <= epochStats.Epoch && epochStats.Epoch < uint64(validator.Validator.ExitEpoch) {
			validatorStats.ValidatorCount++
			validatorStats.ValidatorBalance += uint64(validator.Balance)
			validatorStats.EligibleAmount += uint64(validator.Validator.EffectiveBalance)
		}
	}
	epochStats.validatorStats = validatorStats
}
